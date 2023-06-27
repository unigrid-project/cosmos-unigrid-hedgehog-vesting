package app_test

import (
	"fmt"
	"testing"

	ugdvestingante "github.com/timnhanta/ugdvesting/x/hedgehogvesting/ante"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

const (
	alice    = "cosmos1tz8060hlzpnclakurlkq9v7zwhycgqax8gv04f"
	bob      = "cosmos1xgpj0q8lslnylvu78vkn88duhgp724zlgm79mk"
	balAlice = 1000
	balBob   = 2000
)

type AnteTestSuite struct {
	suite.Suite

	app         *simapp.SimApp
	anteHandler sdk.AnteHandler
	ctx         sdk.Context
	clientCtx   client.Context
	txBuilder   client.TxBuilder
}

// returns context and app with params set on account keeper
func createTestApp(t *testing.T, isCheckTx bool) (*simapp.SimApp, sdk.Context) {
	app := simapp.Setup(t, isCheckTx)
	ctx := app.BaseApp.NewContext(isCheckTx, tmproto.Header{})
	app.AccountKeeper.SetParams(ctx, authtypes.DefaultParams())
	app.BankKeeper.SetParams(ctx, banktypes.DefaultParams())

	return app, ctx
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, new(AnteTestSuite))
}

func (s *AnteTestSuite) SetupTest(isCheckTx bool) {
	s.app, s.ctx = createTestApp(s.T(), isCheckTx)
	s.ctx = s.ctx.WithBlockHeight(1)

	// Set up TxConfig.
	encodingConfig := simapp.MakeTestEncodingConfig()
	// We're using TestMsg encoding in some tests, so register it here.
	encodingConfig.Amino.RegisterConcrete(&testdata.TestMsg{}, "testdata.TestMsg", nil)
	testdata.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	s.clientCtx = client.Context{}.
		WithTxConfig(encodingConfig.TxConfig)

	anteHandler, err := ugdvestingante.NewAnteHandler(
		ante.HandlerOptions{
			AccountKeeper: s.app.AccountKeeper,
			BankKeeper:    s.app.BankKeeper,
		},
	)

	s.Require().NoError(err)

	s.anteHandler = anteHandler
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
}

func (suite *AnteTestSuite) TestInvalidTransaction() {
	suite.SetupTest(true)
	suite.setupSuiteWithBalances()

	// keys and addresses
	_, _, addr1 := generateKeysFromSecret([]byte("secret1"))

	// msg and signatures
	msg := testdata.NewTestMsg(addr1)
	feeAmount := testdata.NewTestFeeAmount()
	gasLimit := testdata.NewTestGasLimit()

	require.NoError(suite.T(), suite.txBuilder.SetMsgs(msg))

	suite.txBuilder.SetFeeAmount(feeAmount)
	suite.txBuilder.SetGasLimit(gasLimit)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{}, []uint64{}, []uint64{}
	invalidTx, err := suite.CreateTestTx(suite.ctx, privs, accNums, accSeqs, suite.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)

	require.NoError(suite.T(), err)

	antehandler := suite.anteHandler
	_, err = antehandler(suite.ctx, invalidTx, false)

	require.ErrorIs(suite.T(), err, sdkerrors.ErrNoSignatures, "Did not error on invalid tx")
}

// Before running this test, make sure that you have `YOUR_ADDRESS` in Vesting GridSpork
// You can use the following request to add your address to Vesting GridSpork

// curl -X PUT https://localhost:52884/gridspork/vesting-storage/YOUR_ADDRESS
// -H "Content-Type: application/json"
// -H "privateKey: YOUR_HEDGEHOG_PRIVATE_KEY"
// -d '{"amount": "16000","start": "2023-06-26T13:32:00.00Z","duration": 60000,"parts": 2}' --insecure

// JSON data can be changed according to your needs
func (suite *AnteTestSuite) TestValidTransaction() {
	suite.SetupTest(true)
	suite.setupSuiteWithBalances()

	// keys and addresses
	_, _, addr1 := generateKeysFromSecret([]byte("secret1"))
	priv2, _, addr2 := generateKeysFromSecret([]byte("secret2"))

	// msg and signatures
	var amount int64 = 5000
	coin := sdk.Coin{Denom: "ugd", Amount: sdk.NewInt(amount)}

	msg := &banktypes.MsgSend{
		FromAddress: addr2.String(),
		ToAddress:   addr1.String(),
		Amount:      sdk.Coins{coin},
	}
	feeAmount := testdata.NewTestFeeAmount()
	gasLimit := testdata.NewTestGasLimit()

	require.NoError(suite.T(), suite.txBuilder.SetMsgs(msg))

	suite.txBuilder.SetFeeAmount(feeAmount)
	suite.txBuilder.SetGasLimit(gasLimit)

	vbd := ugdvestingante.NewValidateBasicDecorator(suite.app.BankKeeper)
	antehandler := sdk.ChainAnteDecorators(vbd)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv2}, []uint64{0}, []uint64{0}
	validTx, err := suite.CreateTestTx(suite.ctx, privs, accNums, accSeqs, suite.ctx.ChainID(), signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(suite.T(), err)

	_, err = antehandler(suite.ctx, validTx, false)
	if err != nil {
		fmt.Println("errr", err)
	}

	require.Nil(suite.T(), err, "ValidateBasicDecorator returned error on valid tx. err: %v", err)
}

// KeyTestPubAddr generates a new secp256k1 keypair.
func generateKeysFromSecret(secret []byte) (cryptotypes.PrivKey, cryptotypes.PubKey, sdk.AccAddress) {
	key := secp256k1.GenPrivKeyFromSecret(secret)
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}

// CreateTestTx is a helper function to create a tx given multiple inputs.
func (suite *AnteTestSuite) CreateTestTx(
	ctx sdk.Context, privs []cryptotypes.PrivKey,
	accNums, accSeqs []uint64,
	chainID string, signMode signing.SignMode,
) (xauthsigning.Tx, error) {
	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	var sigsV2 []signing.SignatureV2
	for i, priv := range privs {
		sigV2 := signing.SignatureV2{
			PubKey: priv.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  signMode,
				Signature: nil,
			},
			Sequence: accSeqs[i],
		}

		sigsV2 = append(sigsV2, sigV2)
	}
	err := suite.txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	// Second round: all signer infos are set, so each signer can sign.
	sigsV2 = []signing.SignatureV2{}
	for i, priv := range privs {
		signerData := xauthsigning.SignerData{
			Address:       sdk.AccAddress(priv.PubKey().Address()).String(),
			ChainID:       chainID,
			AccountNumber: accNums[i],
			Sequence:      accSeqs[i],
			PubKey:        priv.PubKey(),
		}
		sigV2, err := tx.SignWithPrivKey(
			signMode, signerData,
			suite.txBuilder, priv, suite.clientCtx.TxConfig, accSeqs[i])
		if err != nil {
			return nil, err
		}

		sigsV2 = append(sigsV2, sigV2)
	}
	err = suite.txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	return suite.txBuilder.GetTx(), nil
}

func makeBalance(address string, balance int64, denom string) banktypes.Balance {
	return banktypes.Balance{
		Address: address,
		Coins: sdk.Coins{
			sdk.Coin{
				Denom:  denom,
				Amount: sdk.NewInt(balance),
			},
		},
	}
}

func addAll(balances []banktypes.Balance) sdk.Coins {
	total := sdk.NewCoins()
	for _, balance := range balances {
		total = total.Add(balance.Coins...)
	}
	return total
}

func getBankGenesis() *banktypes.GenesisState {
	coins := []banktypes.Balance{
		makeBalance(alice, balAlice, "ugd"),
		makeBalance(bob, balBob, "ugd"),
	}
	supply := banktypes.Supply{
		Total: addAll(coins),
	}

	state := banktypes.NewGenesisState(
		banktypes.DefaultParams(),
		coins,
		supply.Total,
		[]banktypes.Metadata{})

	return state
}

func (suite *AnteTestSuite) setupSuiteWithBalances() {
	suite.app.BankKeeper.InitGenesis(suite.ctx, getBankGenesis())
}
