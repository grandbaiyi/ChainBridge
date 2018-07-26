package main

import (
	"fmt"
	"net/http"
	//"math/big"
	"bytes"
	"io/ioutil"
	"time"
	"jsonparser"
	"encoding/json"
	//"encoding/hex"
	"path/filepath"
	"strings"
	"log"

	//"github.com/ethereum/go-ethereum"
    "github.com/ethereum/go-ethereum/accounts/abi"
)

// used for json format a response from an RPC call
type Resp struct {
	jsonrpc string
	id int
	result string
}

// used to json format an RPC call
type Call struct {
	Jsonrpc string `json:"jsonrpc"`
	Method string `json:"method"`
	Params []string `json:"params"`
	Id int `json:"id"`
}

// used for getLogs json formatting
type LogParams struct {
	FromBlock string `json:"fromBlock"`
}

// this function makes the rpc call "eth_getLogs" passing in jsonParams as the json formatted
// parameters to the call
// json parameters: [optional] fromBlock, toBlock
func getLogs(url string, jsonParams string, client *http.Client) (*http.Response, error) {
	jsonStr := `{"jsonrpc":"2.0","method":"eth_getLogs","params":[` + jsonParams + `],"id":74}`
	jsonBytes := []byte(jsonStr)
	//fmt.Println(string(jsonBytes))

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil { return nil, err }
	return resp, nil
}

// this function makes the rpc call "eth_getTranscationReceipt" passing in the txHash
func getTxReceipt(txHash string, url string, client *http.Client) (*http.Response, error) {
    jsonStr := `{"jsonrpc":"2.0","method":"eth_getTransactionReceipt","params":["` + txHash + `"],"id":74}`
    jsonBytes := []byte(jsonStr)
    //fmt.Println(string(jsonBytes))

    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
    req.Header.Set("Content-Type", "application/json")
    resp, err := client.Do(req)
    if err != nil { return nil, err }
    return resp, nil
}


// this function parses jsonStr for the result entry and returns its value as a string
func parseJsonForResult(jsonStr string) (string, error) {
	jsonBody := []byte(string(jsonStr))
	res, _, _, err := jsonparser.Get(jsonBody, "result")
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// this function parses jsonStr for the entry "get" and returns its value as a string
func parseJsonForEntry(jsonStr string, get string) (string, error) {
	jsonBody := []byte(string(jsonStr))
	res, _, _, err := jsonparser.Get(jsonBody, get)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// this function gets the current block number by calling "eth_blockNumber"
func getBlockNumber(url string, client *http.Client) (string, error) {
	var jsonBytes = []byte(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":83}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	blockNumResp, err := client.Do(req)
	if err != nil {
       	return "", err
	}
	defer blockNumResp.Body.Close()

	// print out response of eth_blockNumber
	//fmt.Println("response Status:", blockNumResp.Status)
	//fmt.Println("response Headers:", blockNumResp.Header)
	blockNumBody, _ := ioutil.ReadAll(blockNumResp.Body)
	//fmt.Println("response Body:", string(blockNumBody))

	// parse json for result
	startBlock, err := parseJsonForResult(string(blockNumBody))
	if err != nil {
		return "", nil
	}
	return startBlock, nil
}

func readDepositData(data string) (string, string) {
	length := len(data)
	if length == 130 { // '0x' + 64 + 64
		recipient := "0x" + data[26:66]
		value := data[66:130]
		return recipient, value
	} else {
		return "", ""
	}
}

func main() {
	// hard coded to client running at address:port
	url := "http://127.0.0.1:8545"
    client := &http.Client{}
	var params LogParams

	path, _ := filepath.Abs("./truffle/build/contracts/Bridge.json")
	file, err := ioutil.ReadFile(path)
	if err != nil {
	    fmt.Println("Failed to read file:", err)
	}

	fileAbi, err := parseJsonForEntry(string(file), "abi")
	if err != nil {
		log.Fatal(err)
	}

	bridgeabi, err := abi.JSON(strings.NewReader(fileAbi))
	if err != nil {
	    fmt.Println("Invalid abi:", err)
	}

	// config file reading
	path, _ = filepath.Abs("./config.json")
	file, err = ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("Failed to read file:", err)	
	}

	homeStr, err := parseJsonForEntry(string(file), "home")
	if err != nil {
		log.Fatal(err)
	}
	homeAddr, err := parseJsonForEntry(homeStr, "contractAddr")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("home contract address: ", homeAddr)

	foreignStr, err := parseJsonForEntry(string(file), "foreign")
	if err != nil {
		log.Fatal(err)
	}
	foreignAddr, err := parseJsonForEntry(foreignStr, "contractAddr")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("foreign contract address: ", foreignAddr, "\n")

	// checking for abi methods
	// bridgeMethods := bridgeabi.Methods
	// transferMethod := bridgeMethods["transfer"]
	// transferSig := transferMethod.Sig()
	// s := string(transferSig[:])
	// fmt.Println(s)

	// checking abi for events
	bridgeEvents := bridgeabi.Events
	depositEvent := bridgeEvents["Deposit"]
	depositHash := depositEvent.Id()
	depositId := depositHash.Hex()
	fmt.Println("deposit event id: ", depositId) // this is the deposit event to watch for

	creationEvent := bridgeEvents["ContractCreation"]
	creationHash := creationEvent.Id()
	creationId := creationHash.Hex()
	fmt.Println("contract creation event id: ", creationId)
	fmt.Println()

	// poll filter every 500ms for changes
	ticker := time.NewTicker(500 * time.Millisecond)
	go func() {
		for t := range ticker.C {
			fmt.Println(t)

			params.FromBlock, _ = getBlockNumber(url, client)
			fmt.Println("getting logs from block number: " + params.FromBlock + "\n")
			jsonParams, _ := json.Marshal(params)
            //fmt.Println("jsonParams: " + string(jsonParams))

			//get logs from params.FromBlock
			resp, _ := getLogs(url, string(jsonParams), client)
			defer resp.Body.Close()

			//fmt.Println("response Status:", resp.Status)
			//fmt.Println("response Headers:", resp.Header)
			body, _ := ioutil.ReadAll(resp.Body)
			//fmt.Println("response Body:", string(body))
 
			// parse for getLogs result
			//logsResult := parseJsonForResult(string(body))
			logsResult, err := parseJsonForEntry(string(body), "result")
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("logsResult: " + logsResult + "\n")
			//fmt.Println(len(logsResult))

			// if there are new logs, parse for event info
			if len(logsResult) != 2 {
				fmt.Println("new logs found")
				//txHash := parseJsonForEntry(logsResult[1:len(logsResult)-1], "transactionHash")
				//fmt.Println(txHash + "\n")

				// get logs contract address
				address, err := parseJsonForEntry(logsResult[1:len(logsResult)-1], "address")
				if err != nil {
					fmt.Println(err)
				}
				// this is not actually a good way to listen for events from a  contract
				// this could be used to confirm a log, but for listening to events from
				// one contract, we would specify the address in our call to eth_getLogs
				fmt.Println("contract addr: ", address)
				fmt.Println("length of address: ", len(address))
				if strings.Compare(address[1:41], homeAddr) == 0 {
					fmt.Println("home bridge contract event heard")
				} else if strings.Compare(address[1:41], foreignAddr) == 0 {
					fmt.Println("foreign bridge contract event heard")
				}

				// read topics of log
				topics,err := parseJsonForEntry(logsResult[1:len(logsResult)-1], "topics")
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println("topics: ", topics[2:68])
				//fmt.Println("length of topics: ", len(topics)-4) len = 66: 0x + 64 hex chars = 32 bytes

				if strings.Compare(topics[2:68],depositId) == 0 { 
					fmt.Println("*** deposit event ", topics[2:68])
					data, err := parseJsonForEntry(logsResult[1:len(logsResult)-1], "data")
					if err != nil {
						fmt.Println(nil)
					}
					fmt.Println("length of data: ", len(data))
					fmt.Println("data: ", data)
					receiver, value := readDepositData(data)
					fmt.Println("receiver: ", receiver) 
					fmt.Println("value: ", value) // in hexidecimal
			 	} else if strings.Compare(topics[2:68],creationId) == 0 {
					fmt.Println("*** bridge contract creation\n")
				}

				fmt.Println()
			}

		}
	}()

	time.Sleep(300 * time.Second)
	ticker.Stop()
}
