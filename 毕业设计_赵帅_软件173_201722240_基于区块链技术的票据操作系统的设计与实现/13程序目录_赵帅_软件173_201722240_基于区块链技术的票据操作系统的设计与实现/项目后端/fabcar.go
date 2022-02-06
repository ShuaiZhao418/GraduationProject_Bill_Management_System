package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

// Personal info struct
type SignInfo struct {
	Username    string `form:"Username" json:"Username"`       // 用户名
	Password    string `form:"Password" json:"Password"`       // 密码
	CompanyName string `form:"CompanyName" json:"CompanyName"` // 公司名称
	CompanyId   string `form:"CompanyId" json:"CompanyId"`     //  公司ID
}

// Bill info struct
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
}

var contract *gateway.Contract

func main() {
	os.Setenv("DISCOVERY_AS_LOCALHOST", "true")
	// 创建一个钱包
	// Build a wallet
	wallet, err := gateway.NewFileSystemWallet("wallet")
	if err != nil {
		fmt.Printf("Failed to create wallet: %s\n", err)
		os.Exit(1)
	}
	// 向钱包中添加用户的证书，生成目录等
	// Add the User Certificate, content... to the wallet
	if !wallet.Exists("appUser2") {
		err = populateWallet(wallet)
		if err != nil {
			fmt.Printf("Failed to populate wallet contents: %s\n", err)
			os.Exit(1)
		}
	}
	// 获取链接配置文件
	// The configuration files
	ccpPath := filepath.Join(
		"..",
		"..",
		"test-network",
		"organizations",
		"peerOrganizations",
		"org2.example.com",
		"connection-org2.yaml",
	)
	// 从钱包中获取身份，创立链接
	gw, err := gateway.Connect(
		gateway.WithConfig(config.FromFile(filepath.Clean(ccpPath))),
		gateway.WithIdentity(wallet, "appUser2"),
	)
	if err != nil {
		fmt.Printf("Failed to connect to gateway: %s\n", err)
		os.Exit(1)
	}
	defer gw.Close()
	// 连接到通道
	// connect to the Blockchain Network
	network, err := gw.GetNetwork("mychannel")
	if err != nil {
		fmt.Printf("Failed to get network: %s\n", err)
		os.Exit(1)
	}
	// 获取链上代码（智能合约）名称
	// Get smart contract name
	contract = network.GetContract("fabcar")

	// 进行初始化
	// Get init
	result, err := contract.SubmitTransaction("initLedger")
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(result))

	router := gin.Default()
	router.Use(Cors()) //开启中间件 允许使用跨域请求
	// 定义路由
	// Routers for admin / bank's / company's functions
	A1 := router.Group("/A1/admin")
	{
		// admin test funtion
		A1.POST("/admintest", admintest)
		// Query all the sign info function
		A1.POST("/queryAllSignInfos", queryAllSignInfos)
	}
	B1 := router.Group("/B1/bank")
	{
		// issue bill function
		// 发布票据
		B1.POST("/issueBill", issueBill)

		// Search all the bill infos
		// 查看所有票据
		B1.POST("/queryAllBills", queryAllBills)

		// Change the bill's pay user
		// 更换承兑人（参加承兑）
		B1.POST("/changePayBillInfo", changePayBillInfo)

		// Search all the bills which are waiting for discounting
		// 查询待贴现票据-查询
		B1.POST("/queryWaitDiscountBills", queryWaitDiscountBills)

		// -处理待贴现票据	 Deal with the bills which are waiting for discounting
		// - 同意贴现   Agree to discount
		B1.POST("/agreeDiscountBills", agreeDiscountBills)

		// - 拒绝贴现	refuse to discount
		B1.POST("/aDisagreeDiscountBills", aDisagreeDiscountBills)

		// 查询历史记录	  Search a bill's operation history
		B1.POST("/queryHistoryById", queryHistoryById)
	}
	C1 := router.Group("/C1/company")
	{
		// 查看待承兑票据  Search all the bills which are waiting for paying
		C1.POST("/checkWaitPayBills", checkWaitPayBills)
		// agree to pay (set bill state to public)
		// 同意承兑 - 将State变为public
		C1.POST("/agreePay", agreePay)
		// refuse to pay (delete this bill's infos except the billID)
		// 拒绝承兑 - 将所有信息抹去，State变为“BillFail”
		C1.POST("/disagreePay", disagreePay)

		// 查看自己的所有票据  Saerch bill infos which are related to this user
		// -查看pay身份的bill	Bills which to check
		C1.POST("/checkAllPayBills", checkAllPayBills)
		// -查看accept身份的bill   Bills which to accept
		C1.POST("/checkAllAcceptBills", checkAllAcceptBills)
		// -查看hold身份的bill   Bills which to hole
		C1.POST("/checkAllHoldBills", checkAllHoldBills)

		// 贴现操作	 Apply to discount bill function
		C1.POST("/discountBill", discountBill)

		// 票据背书	 Apply to endorse bill function
		C1.POST("/endorseBill", endorseBill)
		// 待背书票据	Search all the bills which are waiting for being endorsed
		C1.POST("/checkWaitEndorseBills", checkWaitEndorseBills)
		// 同意背书操作   Agree to endorse
		C1.POST("/agreeEndorseBill", agreeEndorseBill)
		// 拒绝背书操作	  Refuse to endorse
		C1.POST("/disagreeEndorseBill", disagreeEndorseBill)

	}
	// listen port
	router.Run(":8000")
}

//——————————————————————————————管理员———————admin———————————————————————————————————————
// 登陆处理 admin test funtion
func admintest(ctx *gin.Context) {
	// 绑定传来的form
	// Receive the from and get info from the front end
	var signinfo SignInfo
	err := ctx.ShouldBind(&signinfo)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}

	// 调用智能合约中的QuerySignInfo方法，查询所有用户信息
	// Call the QuerySignInfo function, query all the users' infos
	results, err := contract.SubmitTransaction("querySignInfo")
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}

	// Deal with the receive infos
	// 声明SignInfo类型结构体数组
	var jsonresults []SignInfo
	// 将返回数据序列化，存入上述结构体数组 deserialize
	json.Unmarshal(results, &jsonresults)

	// Check the admin infos
	// 遍历查询结果,如果用户名密码符合，则返回其signinfo，否则返回失败
	for i := range jsonresults {
		// 如果数据符合条件，返回此signinfo
		if jsonresults[i].Username == signinfo.Username && jsonresults[i].Password == signinfo.Password {
			fmt.Println(jsonresults[i])
			data, _ := json.Marshal(jsonresults[i])
			ctx.JSON(http.StatusOK, string(data))
		}
	}
}

// 查询所有signinfo   Query all the sign info function
func queryAllSignInfos(ctx *gin.Context) {
	// 进行交易
	results, err := contract.SubmitTransaction("querySignInfo")
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))

	ctx.JSON(http.StatusOK, string(results))
}

//——————————————————————————————银行——————bank——————————————————————————————————————————
// 发布票据  Issue a bill
func issueBill(ctx *gin.Context) {
	// 绑定传来的form
	// receive the bill's info from front-end
	var bill Bill
	err := ctx.ShouldBind(&bill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// Call issueBill smart contract to create the new bill
	// 调用智能合约中的issueBill方法，并传递票据信息
	results, err := contract.SubmitTransaction("issueBill", bill.BillInfoID, bill.BillInfoMoney, bill.BillInfoType, bill.BillInfoIssueDate, bill.BillInfoDueDate, bill.PubBillID, bill.PubBillName, bill.PayBillID, bill.PayBillName, bill.AcceptBillID, bill.AcceptBillName, bill.HoldBillID, bill.HoldBillName)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
}

// 查看所有票据 Query all the bill infos
func queryAllBills(ctx *gin.Context) {
	// Call queryAllBill smart contract to query all the bill infos
	// 向区块链发送交易请求，调用智能合约中的queryAllBill方法，查询票据信息
	results, err := contract.SubmitTransaction("queryAllBill")
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
	// 将查询结果以字符串形式发回前端
	ctx.JSON(http.StatusOK, string(results))
}

// 更换承兑人（参加承兑） change the pay user
func changePayBillInfo(ctx *gin.Context) {
	// Search the bill info by id
	//首先根据id查询票据
	// 绑定传来的form
	var getbill Bill
	err := ctx.ShouldBind(&getbill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 查询此id的票据的信息
	results, err := contract.SubmitTransaction("QueryBillById", getbill.BillInfoID)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
	// Update the bill's infos, change the bill's pay person info
	// 将bill转为struct格式,以便对修改承兑人
	var bill Bill
	json.Unmarshal(results, &bill)
	fmt.Println(bill)
	bill.PayBillID = getbill.PayBillID
	bill.PayBillName = getbill.PayBillName
	// 将修改好承兑人信息的数据存入区块链
	// Call issueBill smart contract to update, it will cover the bill's info with the same bill id
	results, err = contract.SubmitTransaction("issueBill", bill.BillInfoID, bill.BillInfoMoney, bill.BillInfoType, bill.BillInfoIssueDate, bill.BillInfoDueDate, bill.PubBillID, bill.PubBillName, bill.PayBillID, bill.PayBillName, bill.AcceptBillID, bill.AcceptBillName, bill.HoldBillID, bill.HoldBillName)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
}

// 查询待贴现票据  Query all the bills which are waiting for discount
// -查询
func queryWaitDiscountBills(ctx *gin.Context) {
	// 绑定传来的form
	var getbill Bill
	err := ctx.ShouldBind(&getbill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 调用智能合约方法queryWaitDiscountBills，查询所有待贴现票据
	// Call queryWaitDiscountBills() smart contract to query
	results, err := contract.SubmitTransaction("queryWaitDiscountBills")
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
	ctx.JSON(http.StatusOK, string(results))
}

// -处理
// -同意贴现：更改状态为Public且更改Message为DiscountSuccess,并且将 收款方面的信息更改为银行
// Agree to discount,
// 1. Change the bill state to Public
// 2. Change the bill message to DiscountSuccess
// 3. Change the bill's pay user info
func agreeDiscountBills(ctx *gin.Context) {
	// 绑定传来的form
	var bill Bill
	err := ctx.ShouldBind(&bill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 调用智能合约方法agreeDiscountBill，并将新的票据信息传入方法中
	// Call agreeDiscountBill() smart contract to update bill info
	results, err := contract.SubmitTransaction("agreeDiscountBill", bill.BillInfoID, bill.BillInfoMoney, bill.BillInfoType, bill.BillInfoIssueDate, bill.BillInfoDueDate, bill.PubBillID, bill.PubBillName, bill.PayBillID, bill.PayBillName, bill.AcceptBillID, bill.AcceptBillName, bill.HoldBillID, bill.HoldBillName)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
}

// -拒绝贴现：更改状态为Public且更改Message为DiscountFail
// Disagree to discont
// 1. Change the bill state to Public
// 2. Change the bill message to DiscountFail
func aDisagreeDiscountBills(ctx *gin.Context) {
	// 绑定传来的form
	var bill Bill
	err := ctx.ShouldBind(&bill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 进行交易
	results, err := contract.SubmitTransaction("aDisagreeDiscountBill", bill.BillInfoID, bill.BillInfoMoney, bill.BillInfoType, bill.BillInfoIssueDate, bill.BillInfoDueDate, bill.PubBillID, bill.PubBillName, bill.PayBillID, bill.PayBillName, bill.AcceptBillID, bill.AcceptBillName, bill.HoldBillID, bill.HoldBillName)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
}

// Query the bill's operation history
// -查询历史交易记录
func queryHistoryById(ctx *gin.Context) {
	// Query by bill's id, receive the id from front-end
	// 接收票据编号信息
	var getbill Bill
	err := ctx.ShouldBind(&getbill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 调用智能合约queryHistoryById方法，并传递票据编号以查询
	// Call queryHistoryById smart contract to query
	results, err := contract.SubmitTransaction("queryHistoryById", getbill.BillInfoID)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
	ctx.JSON(http.StatusOK, string(results))
}

//——————————————————————————————企业————————company————————————————————————————————————————

// 查看待承兑票据 - 需要查询 PayBillID 为个人 和 State 为 Made 的数据
// Query all the bill's which are waiting for the company to pay
func checkWaitPayBills(ctx *gin.Context) {
	// Receive company's ID
	//接收传来的公司ID
	var getbill Bill
	err := ctx.ShouldBind(&getbill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 调用智能合约queryWaitPayBills，传递公司ID参数。
	// Query by calling queryWaitPayBills() smart contract, according to company's ID and bill state 'Made'
	results, err := contract.SubmitTransaction("queryWaitPayBills", getbill.PayBillID)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
	ctx.JSON(http.StatusOK, string(results))
}

// 同意承兑 - 将State变为public
// Agree to pay, change the bill state to public
func agreePay(ctx *gin.Context) {
	// 绑定传来的form
	var bill Bill
	err := ctx.ShouldBind(&bill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 调用智能合约方法agreePayBill，对票据的状态进行修改
	// Call agreePayBill() smart contract to update bill info
	results, err := contract.SubmitTransaction("agreePayBill", bill.BillInfoID, bill.BillInfoMoney, bill.BillInfoType, bill.BillInfoIssueDate, bill.BillInfoDueDate, bill.PubBillID, bill.PubBillName, bill.PayBillID, bill.PayBillName, bill.AcceptBillID, bill.AcceptBillName, bill.HoldBillID, bill.HoldBillName)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
}

// 拒绝承兑 - 将所有信息抹去，State变为“BillFail”
// Refuse to pay, delete bill's info except the ID
func disagreePay(ctx *gin.Context) {
	// 绑定传来的form
	var bill Bill
	err := ctx.ShouldBind(&bill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 进行交易
	results, err := contract.SubmitTransaction("disagreePayBill", bill.BillInfoID)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
}

// Query all the bills related to this user
// 查看自己的所有票据
// Query bills which need to pay and state is 'Public'
// -查看pay身份的bill且state为Public的
func checkAllPayBills(ctx *gin.Context) {
	//首先根据id查询票据
	// 绑定传来的form
	var getbill Bill
	err := ctx.ShouldBind(&getbill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 进行交易
	results, err := contract.SubmitTransaction("queryAllPayBills", getbill.PayBillID)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))

	ctx.JSON(http.StatusOK, string(results))
}

// -查看accept身份的bill且state为Public的
// Query bills which need to accept and state is 'Public'
func checkAllAcceptBills(ctx *gin.Context) {
	//首先根据id查询票据
	// 绑定传来的form
	var getbill Bill
	err := ctx.ShouldBind(&getbill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 进行交易
	results, err := contract.SubmitTransaction("queryAllAcceptBills", getbill.AcceptBillID)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))

	ctx.JSON(http.StatusOK, string(results))
}

// -查看hold身份的bill且state为Public的
// Query bills which need to hold and state is 'Public'
func checkAllHoldBills(ctx *gin.Context) {
	//首先根据id查询票据
	// 绑定传来的form
	var getbill Bill
	err := ctx.ShouldBind(&getbill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 进行交易
	results, err := contract.SubmitTransaction("queryAllHoldBills", getbill.HoldBillID)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))

	ctx.JSON(http.StatusOK, string(results))
}

// 票据背书 --增加 EndorsedID、EndorsedName ，修改 State 为 EnWaitSign
// Apply to endorse, add bill's EndorsedID、EndorsedName infos and update the state to EnWaitSign
func endorseBill(ctx *gin.Context) {
	// Get the EndorsedID、EndorsedName from front end
	// 获取前端传递的票据被背书人的ID和名称
	var bill Bill
	err := ctx.ShouldBind(&bill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 调用智能合约方法endorseBill，并传递票据的被背书人信息
	// Call endorseBill() smart contract to update
	results, err := contract.SubmitTransaction("endorseBill", bill.BillInfoID, bill.BillInfoMoney, bill.BillInfoType, bill.BillInfoIssueDate, bill.BillInfoDueDate, bill.PubBillID, bill.PubBillName, bill.PayBillID, bill.PayBillName, bill.AcceptBillID, bill.AcceptBillName, bill.HoldBillID, bill.HoldBillName, bill.EndorsedID, bill.EndorsedName)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
}

// 同意票据背书  -- 更换承兑人
// Agree to endorse, update the bill's pay user info
func agreeEndorseBill(ctx *gin.Context) {
	// 绑定传来的form
	var bill Bill
	err := ctx.ShouldBind(&bill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 进行交易
	results, err := contract.SubmitTransaction("agreeEndorseBill", bill.BillInfoID, bill.BillInfoMoney, bill.BillInfoType, bill.BillInfoIssueDate, bill.BillInfoDueDate, bill.PubBillID, bill.PubBillName, bill.PayBillID, bill.PayBillName, bill.AcceptBillID, bill.AcceptBillName, bill.HoldBillID, bill.HoldBillName, bill.EndorsedID, bill.EndorsedName)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
}

// 拒绝票据背书
// Disagree to endorse
func disagreeEndorseBill(ctx *gin.Context) {
	// 绑定传来的form
	var bill Bill
	err := ctx.ShouldBind(&bill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 进行交易
	results, err := contract.SubmitTransaction("disagreeEndorseBill", bill.BillInfoID, bill.BillInfoMoney, bill.BillInfoType, bill.BillInfoIssueDate, bill.BillInfoDueDate, bill.PubBillID, bill.PubBillName, bill.PayBillID, bill.PayBillName, bill.AcceptBillID, bill.AcceptBillName, bill.HoldBillID, bill.HoldBillName)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
}

// 待背书票据 -- 依据 EndorsedID 和 State 为 EnWaitSign 查询
// Query bills which are waiting for endorsement, accoridng to bill's EndorsedID and state
func checkWaitEndorseBills(ctx *gin.Context) {
	// 绑定传来的form
	var getbill Bill
	err := ctx.ShouldBind(&getbill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 进行交易
	results, err := contract.SubmitTransaction("queryWaitEndorseBills", getbill.EndorsedID)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))

	ctx.JSON(http.StatusOK, string(results))
}

// 贴现操作 - 将 State 改为 DcWaitSigned
// Apply to discount, change the state to 'DcWaitSigned'
func discountBill(ctx *gin.Context) {
	// 绑定前端传来的所操作票据的信息
	var bill Bill
	err := ctx.ShouldBind(&bill)
	if err != nil {
		fmt.Printf("绑定失败: %s\n", err)
	} else {
		fmt.Printf("绑定成功: %s\n", err)
	}
	// 调用智能合约中的discountBill方法，修改票据状态
	results, err := contract.SubmitTransaction("discountBill", bill.BillInfoID, bill.BillInfoMoney, bill.BillInfoType, bill.BillInfoIssueDate, bill.BillInfoDueDate, bill.PubBillID, bill.PubBillName, bill.PayBillID, bill.PayBillName, bill.AcceptBillID, bill.AcceptBillName, bill.HoldBillID, bill.HoldBillName)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(string(results))
}

// 构造钱包内部的方法populateWallet
// For building the wallet
func populateWallet(wallet *gateway.Wallet) error {
	credPath := filepath.Join(
		"..",
		"..",
		"test-network",
		"organizations",
		"peerOrganizations",
		"org2.example.com",
		"users",
		"User1@org2.example.com",
		"msp",
	)

	certPath := filepath.Join(credPath, "signcerts", "cert.pem")
	// read the certificate pem
	cert, err := ioutil.ReadFile(filepath.Clean(certPath))
	if err != nil {
		return err
	}

	keyDir := filepath.Join(credPath, "keystore")
	// there's a single file in this dir containing the private key
	files, err := ioutil.ReadDir(keyDir)
	if err != nil {
		return err
	}
	if len(files) != 1 {
		return errors.New("keystore folder should have contain one file")
	}
	keyPath := filepath.Join(keyDir, files[0].Name())
	key, err := ioutil.ReadFile(filepath.Clean(keyPath))
	if err != nil {
		return err
	}

	identity := gateway.NewX509Identity("Org2MSP", string(cert), string(key))

	err = wallet.Put("appUser2", identity)
	if err != nil {
		return err
	}
	return nil
}

// 跨域访问
// Solve CORS Problem
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method               //请求方法
		origin := c.Request.Header.Get("Origin") //请求头部
		var headerKeys []string                  // 声明请求头keys
		for k, _ := range c.Request.Header {
			headerKeys = append(headerKeys, k)
		}
		headerStr := strings.Join(headerKeys, ", ")
		if headerStr != "" {
			headerStr = fmt.Sprintf("access-control-allow-origin, access-control-allow-headers, %s", headerStr)
		} else {
			headerStr = "access-control-allow-origin, access-control-allow-headers"
		}
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Origin", "*")                                       // 这是允许访问所有域
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE") //服务器支持的所有跨域请求的方法,为了避免浏览次请求的多次'预检'请求
			//  header的类型
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session,X_Requested_With,Accept, Origin, Host, Connection, Accept-Encoding, Accept-Language,DNT, X-CustomHeader, Keep-Alive, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma")
			//              允许跨域设置                                                                                                      可以返回其他子段
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers,Cache-Control,Content-Language,Content-Type,Expires,Last-Modified,Pragma,FooBar") // 跨域关键设置 让浏览器可以解析
			c.Header("Access-Control-Max-Age", "172800")                                                                                                                                                           // 缓存请求信息 单位为秒
			c.Header("Access-Control-Allow-Credentials", "false")                                                                                                                                                  //  跨域请求是否需要带cookie信息 默认设置为true
			c.Set("content-type", "application/json")                                                                                                                                                              // 设置返回格式是json
		}

		//放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.JSON(http.StatusOK, "Options Request!")
		}
		// 处理请求
		c.Next() //  处理请求
	}
}
