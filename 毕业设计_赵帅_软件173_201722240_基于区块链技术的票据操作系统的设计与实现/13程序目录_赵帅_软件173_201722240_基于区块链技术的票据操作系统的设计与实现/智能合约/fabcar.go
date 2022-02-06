package main

import (
	"encoding/json"
	"fmt"

	//"time"
	"errors"
	// "strconv"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric/common/flogging"
)

// logger
var chaincodeLogger = flogging.MustGetLogger("EndorseBillChaincode")

const (
	// 票据状态
	// Set 6 state to distinguish different type of bills

	// 做成状态
	// "Made" state, waiting for banks to approve
	BillInfo_State_Made = "made"

	// 交付状态
	// "public" state, a normal bill state without any operation
	BillInfo_State_Public = "public"

	// 等待背书状态
	// "enwaitsign" state, waiting for endorsement
	BillInfo_State_EnWaitSign = "enwaitsign"

	// 等待贴现状态
	// "dcwaitsigned" state, waiting for discounting
	BillInfo_State_DcWaitSigned = "dcwaitsigned"

	// 等待承兑状态
	// "waitpay" state, waiting for paying
	BillInfo_State_WaitPay = "waitpay"

	// 等待承兑失败状态，票据作废
	// "billfail" state, waiting for paying fail
	BillInfo_State_BillFail = "billfail"

	// 票据操作提示信息
	// Set 6 message type to note to user, when succeed or fail for different operations

	// 背书成功/失败
	// Endorse success / fail
	Message_EnSuccess = "endorsesuccess"
	Message_EnFail    = "endorsefail"

	// 贴现成功/失败
	// Discount success / fail
	Message_DcSuccess = "discountsuccess"
	Message_DcFail    = "discountfail"

	// 承兑成功/失败
	// Pay success / fail
	Message_WaitPaySuccess = "waitpaysuccess"
	Message_WaitPayFail    = "waitpayfail"
)

type SmartContract struct {
	contractapi.Contract
}

// 票据 Bill struct
type Bill struct {
	//票据基本信息
	BillInfoID        string `json:"BillInfoID"`        //票据号码  Bill ID
	BillInfoMoney     string `json:"BillInfoMoney"`     //票据金额  Bill Amount
	BillInfoType      string `json:"BillInfoType"`      //票据类型	Bill Type
	BillInfoIssueDate string `json:"BillInfoIssueDate"` //票据出票日期  Bill issue date
	BillInfoDueDate   string `json:"BillInfoDueDate"`   //票据到期日期  Bill due date
	//出票人信息  People info (who public this bill)
	PubBillID   string `json:"PubBillID"`   //出票人证件号码  Personal ID
	PubBillName string `json:"PubBillName"` //出票人名称	Personal Name
	//承兑人信息  People info (who pay for this bill)
	PayBillID   string `json:"PayBillID"`   //承兑人证件号码  Personal ID
	PayBillName string `json:"PayBillName"` //承兑人名称  Personal Name
	//收款人信息  People info (who receive the money)
	AcceptBillID   string `json:"AcceptBillID"`   //收款人证件号码  Personal ID
	AcceptBillName string `json:"AcceptBillName"` //收款人名称  Personal Name
	//持票人信息  People info (who own the bill)
	HoldBillID   string `json:"HoldBillID"`   //持票人证件号码  Personal ID
	HoldBillName string `json:"HoldBillName"` //持票人名称  Personal Name
	//背书操作--信息	Attributes for endorsement
	EndorsedID   string `json:"EndorsedID"`   // 被背书人证件号码   Personal ID
	EndorsedName string `json:"EndorsedName"` // 被背书人名称  Personal Name
	Message      string `json:"Message"`      // 操作提示信息  Message for User
	State        string `json:"State"`        //票据状态  Bill State

	// Biil Operation History
	// History []HistoryItem `json:"History"`   //背书历史
	// From the Blockchain feature (immutable), we can all the infos of changes for this bill
	// We do not need to set an attribute, we can search by call the smart contract
}

// 登陆信息数据
// SignInfo struct
type SignInfo struct {
	Username    string `json:"Username"`    // 用户名
	Password    string `json:"Password"`    // 密码
	CompanyName string `json:"CompanyName"` // 公司名称
	CompanyId   string `json:"CompanyId"`   //  公司ID
}

func isExisted(ctx contractapi.TransactionContextInterface, key string) bool {
	val, err := ctx.GetStub().GetState(key)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
	if len(val) == 0 {
		return false
	}
	return true
}

// InitLedger adds a base set of Bills to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	bills := []Bill{
		Bill{BillInfoID: "POA10000998", BillInfoMoney: "2000", BillInfoType: "A", BillInfoIssueDate: "2021-01-20", BillInfoDueDate: "2022-01-20", PubBillID: "bank", PubBillName: "银行", PayBillID: "ccmid", PayBillName: "C公司", AcceptBillID: "acmid", AcceptBillName: "A公司", HoldBillID: "acmid", HoldBillName: "A公司", EndorsedID: "", EndorsedName: "", Message: "", State: "public"},
		Bill{BillInfoID: "POB10000998", BillInfoMoney: "3000", BillInfoType: "B", BillInfoIssueDate: "2021-02-10", BillInfoDueDate: "2022-01-20", PubBillID: "bank", PubBillName: "银行", PayBillID: "acmid", PayBillName: "A公司", AcceptBillID: "bcmid", AcceptBillName: "B公司", HoldBillID: "bcmid", HoldBillName: "B公司", EndorsedID: "", EndorsedName: "", Message: "", State: "public"},
		Bill{BillInfoID: "POC10000998", BillInfoMoney: "40000", BillInfoType: "C", BillInfoIssueDate: "2020-08-23", BillInfoDueDate: "2022-01-20", PubBillID: "bank", PubBillName: "银行", PayBillID: "bmcid", PayBillName: "B公司", AcceptBillID: "ccmid", AcceptBillName: "C公司", HoldBillID: "ccmid", HoldBillName: "C公司", EndorsedID: "", EndorsedName: "", Message: "", State: "public"},
	}

	for _, bill := range bills {
		billAsBytes, _ := json.Marshal(bill)
		err := ctx.GetStub().PutState(bill.BillInfoID, billAsBytes)

		if err != nil {
			return fmt.Errorf("Failed to put to init. %s", err.Error())
		}
	}

	// This is the init function for store some infos to the Blockchain Network at the beginning of running this system
	// By using this, we store the Users' info to the Blockchain Network, which help us to finish the admin and authorization function.
	SignInfos := []SignInfo{
		SignInfo{Username: "admin", Password: "123456", CompanyName: "管理员", CompanyId: "bank"},
		SignInfo{Username: "alice", Password: "123456", CompanyName: "A公司", CompanyId: "acmid"},
		SignInfo{Username: "bob", Password: "123456", CompanyName: "B公司", CompanyId: "bcmid"},
		SignInfo{Username: "carle", Password: "123456", CompanyName: "C公司", CompanyId: "ccmid"},
	}

	for _, signinfo := range SignInfos {
		signinfoAsBytes, _ := json.Marshal(signinfo)
		err := ctx.GetStub().PutState(signinfo.Username, signinfoAsBytes)
		if err != nil {
			return fmt.Errorf("Failed to put to init. %s", err.Error())
		}
	}

	return nil
}

// 登录信息提取
// Quert User's infos
func (s *SmartContract) QuerySignInfo(ctx contractapi.TransactionContextInterface) ([]SignInfo, error) {
	// 根据范围查询，查询系统中所有的用户信息
	// Set the search rage by StartKey and endKey to distinguish among personal infos and bill infos
	startKey := ""
	endKey := ""
	resultsIterator, err := ctx.GetStub().GetStateByRange(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()
	// 声明SignInfo类型结构体数组，接收返回数据
	// Receive the infos
	var results []SignInfo
	// 迭代上述查询到的结果，处理其中的信息
	// use Iterator to deal with infos
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var signinfo SignInfo
		// 将其中一条数据反序列化，存入SignInfo类型变量中
		// deserialize
		err = json.Unmarshal(queryResponse.Value, &signinfo)
		if err != nil {
			return nil, err
		}
		// 将存好一条信息的SignInfo类型变量加入SignInfo类型结构体数组
		// Append to the result when finishing dealing with a bill
		results = append(results, signinfo)
	}
	// 返回查询到的结果
	return results, nil
}

// 查询所有票据信息 Search all the bill infos
func (s *SmartContract) QueryAllBill(ctx contractapi.TransactionContextInterface) ([]Bill, error) {
	// 根据范围查询，查询系统中所有的票据信息
	// Set the search rage by StartKey and endKey to distinguish among personal infos and bill infos
	startKey := ""
	endKey := ""
	resultsIterator, err := ctx.GetStub().GetStateByRange(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()
	// 声明Bill类型结构体数组，接收返回数据
	var results []Bill
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var bill Bill
		err = json.Unmarshal(queryResponse.Value, &bill)
		if err != nil {
			return nil, err
		}
		results = append(results, bill)
	}

	return results, nil
}

// 票据发布 Issue Bill function
// args: 0 - {Bill Object}
func (s *SmartContract) IssueBill(ctx contractapi.TransactionContextInterface, billInfoID string, billInfoMoney string, billInfoType string, billInfoIssueDate string, billInfoDueDate string, pubBillID string, pubBillName string, payBillID string, payBillName string, acceptBillID string, acceptBillName string, holdBillID string, holdBillName string) error {

	//将参数包装为Bill结构体类型
	// Receive the parameters and fill into a bill object
	// Convert the object to json and store to the BlockChain Network
	bill := Bill{
		BillInfoID:        billInfoID,
		BillInfoMoney:     billInfoMoney,
		BillInfoType:      billInfoType,
		BillInfoIssueDate: billInfoIssueDate,
		BillInfoDueDate:   billInfoDueDate,
		PubBillID:         pubBillID,
		PubBillName:       pubBillName,
		PayBillID:         payBillID,
		PayBillName:       payBillName,
		AcceptBillID:      acceptBillID,
		AcceptBillName:    acceptBillName,
		HoldBillID:        holdBillID,
		HoldBillName:      holdBillName,
		EndorsedID:        "",
		EndorsedName:      "",
		Message:           "",
		State:             "made",
	}
	//结构体转json
	// Convert from struct to json
	billAsBytes, _ := json.Marshal(bill)
	// 以票据编号作为key值，将票据信息存入区块链中
	// store bill (BillInfoID as the primary key)
	return ctx.GetStub().PutState(bill.BillInfoID, billAsBytes)
}

// 同意承兑 Agree to pay Function, update the state and messgae of the bill
func (s *SmartContract) AgreePayBill(ctx contractapi.TransactionContextInterface, billInfoID string, billInfoMoney string, billInfoType string, billInfoIssueDate string, billInfoDueDate string, pubBillID string, pubBillName string, payBillID string, payBillName string, acceptBillID string, acceptBillName string, holdBillID string, holdBillName string) error {

	bill := Bill{
		BillInfoID:        billInfoID,
		BillInfoMoney:     billInfoMoney,
		BillInfoType:      billInfoType,
		BillInfoIssueDate: billInfoIssueDate,
		BillInfoDueDate:   billInfoDueDate,
		PubBillID:         pubBillID,
		PubBillName:       pubBillName,
		PayBillID:         payBillID,
		PayBillName:       payBillName,
		AcceptBillID:      acceptBillID,
		AcceptBillName:    acceptBillName,
		HoldBillID:        holdBillID,
		HoldBillName:      holdBillName,
		EndorsedID:        "",
		EndorsedName:      "",
		Message:           "waitpaysuccess",
		State:             "public",
	}
	//结构体转json
	// Convert from struct to json
	billAsBytes, _ := json.Marshal(bill)

	return ctx.GetStub().PutState(bill.BillInfoID, billAsBytes)
}

// 拒绝承兑 Disagree to pay Function, update all the info except billID of the bill
func (s *SmartContract) DisagreePayBill(ctx contractapi.TransactionContextInterface, billInfoID string) error {

	bill := Bill{
		BillInfoID:        billInfoID,
		BillInfoMoney:     "",
		BillInfoType:      "",
		BillInfoIssueDate: "",
		BillInfoDueDate:   "",
		PubBillID:         "",
		PubBillName:       "",
		PayBillID:         "",
		PayBillName:       "",
		AcceptBillID:      "",
		AcceptBillName:    "",
		HoldBillID:        "",
		HoldBillName:      "",
		EndorsedID:        "",
		EndorsedName:      "",
		Message:           "waitpayfail",
		State:             "billfail",
	}
	//结构体转json
	billAsBytes, _ := json.Marshal(bill)

	return ctx.GetStub().PutState(bill.BillInfoID, billAsBytes)
}

// 申请贴现	Apply to discount function (change the state of the bill)
func (s *SmartContract) DiscountBill(ctx contractapi.TransactionContextInterface, billInfoID string, billInfoMoney string, billInfoType string, billInfoIssueDate string, billInfoDueDate string, pubBillID string, pubBillName string, payBillID string, payBillName string, acceptBillID string, acceptBillName string, holdBillID string, holdBillName string) error {

	bill := Bill{
		BillInfoID:        billInfoID,
		BillInfoMoney:     billInfoMoney,
		BillInfoType:      billInfoType,
		BillInfoIssueDate: billInfoIssueDate,
		BillInfoDueDate:   billInfoDueDate,
		PubBillID:         pubBillID,
		PubBillName:       pubBillName,
		PayBillID:         payBillID,
		PayBillName:       payBillName,
		AcceptBillID:      acceptBillID,
		AcceptBillName:    acceptBillName,
		HoldBillID:        holdBillID,
		HoldBillName:      holdBillName,
		EndorsedID:        "",
		EndorsedName:      "",
		Message:           "",
		State:             "dcwaitsigned",
	}
	//结构体转json
	billAsBytes, _ := json.Marshal(bill)

	return ctx.GetStub().PutState(bill.BillInfoID, billAsBytes)
}

// 同意贴现
// Agree to discount,
// 1. Change the bill state to Public
// 2. Change the bill message to DiscountSuccess
// 3. Change the bill's pay user info
func (s *SmartContract) AgreeDiscountBill(ctx contractapi.TransactionContextInterface, billInfoID string, billInfoMoney string, billInfoType string, billInfoIssueDate string, billInfoDueDate string, pubBillID string, pubBillName string, payBillID string, payBillName string, acceptBillID string, acceptBillName string, holdBillID string, holdBillName string) error {

	bill := Bill{
		BillInfoID:        billInfoID,
		BillInfoMoney:     billInfoMoney,
		BillInfoType:      billInfoType,
		BillInfoIssueDate: billInfoIssueDate,
		BillInfoDueDate:   billInfoDueDate,
		PubBillID:         pubBillID,
		PubBillName:       pubBillName,
		PayBillID:         payBillID,
		PayBillName:       payBillName,
		AcceptBillID:      acceptBillID,
		AcceptBillName:    acceptBillName,
		HoldBillID:        holdBillID,
		HoldBillName:      holdBillName,
		EndorsedID:        "",
		EndorsedName:      "",
		Message:           "discountsuccess",
		State:             "public",
	}
	//结构体转json
	billAsBytes, _ := json.Marshal(bill)

	return ctx.GetStub().PutState(bill.BillInfoID, billAsBytes)
}

// 拒绝贴现
// Disagree to discont
// 1. Change the bill state to Public
// 2. Change the bill message to DiscountFail
func (s *SmartContract) ADisagreeDiscountBill(ctx contractapi.TransactionContextInterface, billInfoID string, billInfoMoney string, billInfoType string, billInfoIssueDate string, billInfoDueDate string, pubBillID string, pubBillName string, payBillID string, payBillName string, acceptBillID string, acceptBillName string, holdBillID string, holdBillName string) error {

	bill := Bill{
		BillInfoID:        billInfoID,
		BillInfoMoney:     billInfoMoney,
		BillInfoType:      billInfoType,
		BillInfoIssueDate: billInfoIssueDate,
		BillInfoDueDate:   billInfoDueDate,
		PubBillID:         pubBillID,
		PubBillName:       pubBillName,
		PayBillID:         payBillID,
		PayBillName:       payBillName,
		AcceptBillID:      acceptBillID,
		AcceptBillName:    acceptBillName,
		HoldBillID:        holdBillID,
		HoldBillName:      holdBillName,
		EndorsedID:        "",
		EndorsedName:      "",
		Message:           "discountfail",
		State:             "public",
	}
	//结构体转json
	billAsBytes, _ := json.Marshal(bill)

	return ctx.GetStub().PutState(bill.BillInfoID, billAsBytes)
}

// 申请背书
// Apply to endorse, add bill's EndorsedID、EndorsedName infos and update the state to EnWaitSign
func (s *SmartContract) EndorseBill(ctx contractapi.TransactionContextInterface, billInfoID string, billInfoMoney string, billInfoType string, billInfoIssueDate string, billInfoDueDate string, pubBillID string, pubBillName string, payBillID string, payBillName string, acceptBillID string, acceptBillName string, holdBillID string, holdBillName string, endorsedID string, endorsedName string) error {

	bill := Bill{
		BillInfoID:        billInfoID,
		BillInfoMoney:     billInfoMoney,
		BillInfoType:      billInfoType,
		BillInfoIssueDate: billInfoIssueDate,
		BillInfoDueDate:   billInfoDueDate,
		PubBillID:         pubBillID,
		PubBillName:       pubBillName,
		PayBillID:         payBillID,
		PayBillName:       payBillName,
		AcceptBillID:      acceptBillID,
		AcceptBillName:    acceptBillName,
		HoldBillID:        holdBillID,
		HoldBillName:      holdBillName,
		EndorsedID:        endorsedID,
		EndorsedName:      endorsedName,
		Message:           "",
		State:             "enwaitsign",
	}
	//结构体转json
	billAsBytes, _ := json.Marshal(bill)

	return ctx.GetStub().PutState(bill.BillInfoID, billAsBytes)
}

// 同意背书
// Agree to endorse, update the bill's pay user info
func (s *SmartContract) AgreeEndorseBill(ctx contractapi.TransactionContextInterface, billInfoID string, billInfoMoney string, billInfoType string, billInfoIssueDate string, billInfoDueDate string, pubBillID string, pubBillName string, payBillID string, payBillName string, acceptBillID string, acceptBillName string, holdBillID string, holdBillName string, endorsedID string, endorsedName string) error {

	bill := Bill{
		BillInfoID:        billInfoID,
		BillInfoMoney:     billInfoMoney,
		BillInfoType:      billInfoType,
		BillInfoIssueDate: billInfoIssueDate,
		BillInfoDueDate:   billInfoDueDate,
		PubBillID:         pubBillID,
		PubBillName:       pubBillName,
		PayBillID:         payBillID,
		PayBillName:       payBillName,
		AcceptBillID:      endorsedID,
		AcceptBillName:    endorsedName,
		HoldBillID:        endorsedID,
		HoldBillName:      endorsedName,
		EndorsedID:        "",
		EndorsedName:      "",
		Message:           "endorsesuccess",
		State:             "public",
	}
	//结构体转json
	billAsBytes, _ := json.Marshal(bill)

	return ctx.GetStub().PutState(bill.BillInfoID, billAsBytes)
}

// 拒绝背书
// Disagree to endorse
func (s *SmartContract) DisagreeEndorseBill(ctx contractapi.TransactionContextInterface, billInfoID string, billInfoMoney string, billInfoType string, billInfoIssueDate string, billInfoDueDate string, pubBillID string, pubBillName string, payBillID string, payBillName string, acceptBillID string, acceptBillName string, holdBillID string, holdBillName string) error {

	bill := Bill{
		BillInfoID:        billInfoID,
		BillInfoMoney:     billInfoMoney,
		BillInfoType:      billInfoType,
		BillInfoIssueDate: billInfoIssueDate,
		BillInfoDueDate:   billInfoDueDate,
		PubBillID:         pubBillID,
		PubBillName:       pubBillName,
		PayBillID:         payBillID,
		PayBillName:       payBillName,
		AcceptBillID:      acceptBillID,
		AcceptBillName:    acceptBillName,
		HoldBillID:        holdBillID,
		HoldBillName:      holdBillName,
		EndorsedID:        "",
		EndorsedName:      "",
		Message:           "endorsefail",
		State:             "public",
	}
	//结构体转json
	billAsBytes, _ := json.Marshal(bill)

	return ctx.GetStub().PutState(bill.BillInfoID, billAsBytes)
}

// 根据id查询bill的历史记录
// Query the bill's operation history
func (s *SmartContract) QueryHistoryById(ctx contractapi.TransactionContextInterface, billInfoID string) ([]Bill, error) {

	resultsIterator, err := ctx.GetStub().GetHistoryForKey(billInfoID)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var results []Bill

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var bill Bill
		err = json.Unmarshal(queryResponse.Value, &bill)
		if err != nil {
			return nil, err
		}
		results = append(results, bill)
	}

	return results, nil
}

// 根据id查询bill
// Query bill by ID
func (s *SmartContract) QueryBillById(ctx contractapi.TransactionContextInterface, id string) (*Bill, error) {

	billAsBytes, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("Failed to read from world state. %s", err.Error())
	}
	if billAsBytes == nil {
		return nil, fmt.Errorf("%s does not exist", id)
	}

	bill := new(Bill)
	_ = json.Unmarshal(billAsBytes, bill)

	return bill, nil
}

//条件查询\ 查询state为DcWaitSigned 等待被贴现签收的所有票据
// Query the bills waiting for discounting, accordin to state 'DcWaitSigned'
func (s *SmartContract) QueryWaitDiscountBills(ctx contractapi.TransactionContextInterface) ([]Bill, error) {

	//拼接查询字符串
	// Build a string for search condition
	queryString := fmt.Sprintf("{\"selector\":{\"State\":\"dcwaitsigned\"}}")
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var results []Bill

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var bill Bill
		err = json.Unmarshal(queryResponse.Value, &bill)
		if err != nil {
			return nil, err
		}
		results = append(results, bill)
	}
	return results, nil
}

//条件查询\ 需要查询 PayBillID 为个人 和 State 为 Made 的等待被承兑签收的所有票据
//  Query all the bill's which are waiting for the company to pay, according to company's ID and bill state 'Made'
func (s *SmartContract) QueryWaitPayBills(ctx contractapi.TransactionContextInterface, payBillID string) ([]Bill, error) {
	//获取发起方的公司名
	PayBillID := strings.ToLower(payBillID)
	//拼接查询字符串
	queryString := fmt.Sprintf("{\"selector\":{\"PayBillID\":\"%s\",\"State\":\"made\"}}", PayBillID)
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var results []Bill

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var bill Bill
		err = json.Unmarshal(queryResponse.Value, &bill)
		if err != nil {
			return nil, err
		}
		results = append(results, bill)
	}
	return results, nil
}

//条件查询\-查看pay身份的bill且state为Public的票据
// Query bills which need to pay and state is 'Public'
func (s *SmartContract) QueryAllPayBills(ctx contractapi.TransactionContextInterface, payBillID string) ([]Bill, error) {

	// 拼接查询字符串，根据传入的PayBillID参数和票据状态查询符合要求的票据
	queryString := fmt.Sprintf("{\"selector\":{\"PayBillID\":\"%s\", \"State\":\"public\"}}", payBillID)
	// 将查询字符串传入查询方法
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()
	// 用以接收查询结果
	var results []Bill
	// 遍历返回数据，将其加入Bill数组中。
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var bill Bill
		err = json.Unmarshal(queryResponse.Value, &bill)
		if err != nil {
			return nil, err
		}
		results = append(results, bill)
	}
	return results, nil
}

//条件查询\-查看accept身份的bill且state为Public的票据
// Query bills which need to accept and state is 'Public'
func (s *SmartContract) QueryAllAcceptBills(ctx contractapi.TransactionContextInterface, acceptBillID string) ([]Bill, error) {

	// 拼接查询字符串
	queryString := fmt.Sprintf("{\"selector\":{\"AcceptBillID\":\"%s\", \"State\":\"public\"}}", acceptBillID)
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var results []Bill

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var bill Bill
		err = json.Unmarshal(queryResponse.Value, &bill)
		if err != nil {
			return nil, err
		}
		results = append(results, bill)
	}
	return results, nil
}

//条件查询\-查看hold身份的bill且state为Public的票据
// Query bills which need to hold and state is 'Public'
func (s *SmartContract) QueryAllHoldBills(ctx contractapi.TransactionContextInterface, holdBillID string) ([]Bill, error) {

	// 拼接查询字符串
	queryString := fmt.Sprintf("{\"selector\":{\"HoldBillID\":\"%s\", \"State\":\"public\"}}", holdBillID)
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var results []Bill

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var bill Bill
		err = json.Unmarshal(queryResponse.Value, &bill)
		if err != nil {
			return nil, err
		}
		results = append(results, bill)
	}
	return results, nil
}

//条件查询\-依据 EndorsedID 和 State 为 EnWaitSign 查询
// Query the bill waiting for signed to endorse
func (s *SmartContract) QueryWaitEndorseBills(ctx contractapi.TransactionContextInterface, endorsedID string) ([]Bill, error) {

	// 拼接查询字符串
	queryString := fmt.Sprintf("{\"selector\":{\"EndorsedID\":\"%s\", \"State\":\"enwaitsign\"}}", endorsedID)
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var results []Bill

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var bill Bill
		err = json.Unmarshal(queryResponse.Value, &bill)
		if err != nil {
			return nil, err
		}
		results = append(results, bill)
	}
	return results, nil
}

// 贴现/背书请求 -- 更改收款人和持票人
//  Apply to discount and endorse, change the info of Accept and hold user infos
func (s *SmartContract) DiscountAndEndorse(ctx contractapi.TransactionContextInterface, args []string) error {
	if len(args) != 5 {
		return errors.New("DiscountAndEndorse-参数出错")
	}
	// 以Bill的id查询票据
	billid := args[0]
	billinfo, err := ctx.GetStub().GetState(billid)
	var bill Bill
	err = json.Unmarshal([]byte(billinfo), &bill)
	// 如果查询错误或为空，则说明账号错误
	if err != nil {
		return err
	}
	// 修改bill的收款人和持票人
	bill.AcceptBillID = args[1]
	bill.AcceptBillName = args[2]
	bill.HoldBillID = args[3]
	bill.HoldBillName = args[4]
	// 将修改完的bill存储回区块链
	jsonString, err := json.Marshal(bill)
	fmt.Println("json:" + string(jsonString))
	if err != nil {
		return errors.New("DiscountAndEndorse-json序列化失败")
	}
	return ctx.GetStub().PutState(billid, []byte(jsonString))
}

// 参加承兑 -- 更改承兑人
// Chaneg the bill's accept infos
func (s *SmartContract) ChangeAccept(ctx contractapi.TransactionContextInterface, args []string) error {
	if len(args) != 3 {
		return errors.New("ChangeAccept-参数出错")
	}
	// 以Bill的id查询票据
	billid := args[0]
	billinfo, err := ctx.GetStub().GetState(billid)
	var bill Bill
	err = json.Unmarshal([]byte(billinfo), &bill)
	// 如果查询错误或为空，则说明账号错误
	if err != nil {
		return err
	}
	// 修改bill的收款人和持票人
	bill.PayBillID = args[1]
	bill.PayBillName = args[2]
	// 将修改完的bill存储回区块链
	jsonString, err := json.Marshal(bill)
	fmt.Println("json:" + string(jsonString))
	if err != nil {
		return errors.New("ChangeAccept-json序列化失败")
	}
	return ctx.GetStub().PutState(billid, []byte(jsonString))

}

// 由“做成” 到 “交付”  即更改票据状态 state. 由id查询，
//  From make to public, bill is signed to pay
func (s *SmartContract) ChangeState(ctx contractapi.TransactionContextInterface, args []string) error {
	if len(args) != 2 {
		return errors.New("ChangeState-参数出错")
	}
	// 以Bill的id查询票据
	billid := args[0]
	billinfo, err := ctx.GetStub().GetState(billid)
	var bill Bill
	err = json.Unmarshal([]byte(billinfo), &bill)
	// 如果查询错误或为空，则说明账号错误
	if err != nil {
		return err
	}
	// 修改bill的State
	bill.State = args[1]
	// 将修改完的bill存储回区块链
	jsonString, err := json.Marshal(bill)
	fmt.Println("json:" + string(jsonString))
	if err != nil {
		return errors.New("ChangeState-json序列化失败")
	}
	return ctx.GetStub().PutState(billid, []byte(jsonString))
}

// 更改操作信息提示 message
// Change bill's message info
func (s *SmartContract) ChangeMessage(ctx contractapi.TransactionContextInterface, args []string) error {
	if len(args) != 2 {
		return errors.New("ChangeMessage-参数出错")
	}
	// 以Bill的id查询票据
	billid := args[0]
	billinfo, err := ctx.GetStub().GetState(billid)
	var bill Bill
	err = json.Unmarshal([]byte(billinfo), &bill)
	// 如果查询错误或为空，则说明账号错误
	if err != nil {
		return err
	}
	// 修改bill的Message
	bill.Message = args[1]
	// 将修改完的bill存储回区块链
	jsonString, err := json.Marshal(bill)
	fmt.Println("json:" + string(jsonString))
	if err != nil {
		return errors.New("ChangeMessage-json序列化失败")
	}
	return ctx.GetStub().PutState(billid, []byte(jsonString))
}

// 查询个人票据信息 id (已经承兑)
// Search all the bills' infos which have been paid
func (s *SmartContract) QueryMyBillByIdAndPay(ctx contractapi.TransactionContextInterface, userid string) ([]Bill, error) {
	uid := userid
	state := "Public"
	kk, _ := ctx.GetStub().CreateCompositeKey("searchkey", []string{uid, state})

	resultsIterator, err := ctx.GetStub().GetHistoryForKey(kk)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	results := []Bill{}

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		bill := Bill{}
		_ = json.Unmarshal(queryResponse.Value, bill)
		results = append(results, bill)
	}

	return results, nil
}

// 查询个人票据信息 id (未承兑)
// Search all the bills' infos which have not been paid
func (s *SmartContract) QueryMyBillByIdAndUnpay(ctx contractapi.TransactionContextInterface, userid string) ([]Bill, error) {
	uid := userid
	state := "Made"
	kk, _ := ctx.GetStub().CreateCompositeKey("searchkey", []string{uid, state})

	resultsIterator, err := ctx.GetStub().GetHistoryForKey(kk)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	results := []Bill{}

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		bill := Bill{}
		_ = json.Unmarshal(queryResponse.Value, bill)
		results = append(results, bill)
	}

	return results, nil
}

// 查询票据的历史信息
// 查询所有票据信息 all
// Query bill's operation history
func (s *SmartContract) QueryBillHistoryById(ctx contractapi.TransactionContextInterface, billid string) ([]Bill, error) {
	// 通过bill的id查询bill的历史记录
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(billid)
	if err != nil {
		return nil, errors.New("QueryBillHistoryById-查询历史记录失败")
	}
	defer resultsIterator.Close()

	results := []Bill{}

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		bill := Bill{}
		_ = json.Unmarshal(queryResponse.Value, bill)
		results = append(results, bill)
	}

	return results, nil
}

func main() {

	chaincode, err := contractapi.NewChaincode(new(SmartContract))

	if err != nil {
		fmt.Printf("Error create fabcar chaincode: %s", err.Error())
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting fabcar chaincode: %s", err.Error())
	}
}
