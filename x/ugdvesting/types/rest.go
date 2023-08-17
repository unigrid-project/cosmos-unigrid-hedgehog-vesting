package types

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const (
	HedgehogVestingUrl = "https://localhost:52884/gridspork/vesting-storage/"
	HedgehogMintingUrl = "https://localhost:52884/gridspork/mint-storage"
)

func HegdehogRequestGetVestingByAddr(addr string) *Vesting {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	resp, err := client.Get(HedgehogVestingUrl + addr)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if len(body) == 0 {
		return nil
	}

	var vesting *Vesting

	err = json.Unmarshal([]byte(body), &vesting)
	if err != nil {
		panic(err)
	}

	return vesting
}

type Mints struct {
	Mints map[string]int
}

type HedgehogData struct {
	Data         Mints `json:"data"`
	PreviousData Mints `json:"previousData"`
}

func HegdehogCheckIfInMintingList(addr string) bool {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	resp, err := client.Get(HedgehogMintingUrl)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if len(body) == 0 {
		return false
	}

	var data *HedgehogData

	err = json.Unmarshal([]byte(body), &data)
	if err != nil {
		panic(err)
	}

	for key := range data.Data.Mints {
		if strings.Contains(key, addr) {
			return true
		}
	}

	return false
}
