
// 智能合约

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type OptCode int

const (
	ACCMOUNT = 1000 + iota
	USEDElECTRIC
	CREATELECTRIC
)

type User struct {
	UID string `json:"uid"`
}

//账户基本信息
type AccountInfo struct {
	User
	Side       string `json:"side"`
	Role       string `json:"role"`
	SumEs      int64  `json:"sumEs"`  //当前累计电量
	UsedEs     int64  `json:"usedEs"` //当前消耗电量，发电方为被使用多少电量
	Amount     int64  `json:"amount"`
	UsedAmount int64  `json:"usedAmount"` //实际消耗金额
}

//初始化账户
func (acc *AccountInfo) InitAcc(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	fmt.Println("initAcc args : " + args[0])

	errs := json.Unmarshal([]byte(args[0]), &acc)
	if nil != errs {
		return shim.Error("initAcc args  Incorrect json comvert: " + args[0])
	}
	//初始化账户业务key
	accKey := acc.UID + strconv.Itoa(ACCMOUNT)

	accbytes, jsonErr := json.Marshal(&acc)
	if nil != jsonErr {
		return shim.Error("initAcc struct to json,Incorrect json comvert")
	}
	accErr := stub.PutState(accKey, accbytes)
	if accErr != nil {
		return shim.Error("error:" + accErr.Error())
	}
	return shim.Success(nil)
}

//查询单个账户
func (acc *AccountInfo) QueryAcc(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	fmt.Println("queryAcc args : " + args[0])
	accKey := args[0] + strconv.Itoa(ACCMOUNT)

	accResultBytes, err := stub.GetState(accKey)

	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + args[0] + "\"}"
		return shim.Error(jsonResp)
	}

	if accResultBytes == nil {
		jsonResp := "{\"Error\":\"Nil amount for " + accKey + ":" + string(accResultBytes) + "\"}"
		return shim.Error(jsonResp)
	}
	return shim.Success(accResultBytes)
}

//用电方交易信息
type UsedElectric struct {
	User
	//用电量
	UsedElectric int64 `json:"usedElectric"`
	//电价
	Price int64 `json:"price"`
	//用电开始时间
	UseTimeStart string `json:"useTimeStart"`
	//用电结束时间
	UseTimeEnd string `json:"useTimeEnd"`
}
//结算
type Settlements struct{

	SellUid string `json:"sellUid"`
	BuyUid string `json:"buyUid"`
	SellRole string `json:"sellRole"`
	//电价
	Price int64 `json:"price"`
	Eletric int64 `json:"Eletric"`

}

//结算电价
func (settlements *Settlements) SettlementsElectric(stub shim.ChaincodeStubInterface, args []string) pb.Response {


	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	fmt.Println("invokeUsedElectric args : " + args[0])

	errs := json.Unmarshal([]byte(args[0]), &settlements)
	if nil != errs {
		return shim.Error("settlementsElectric args  Incorrect json comvert: " + args[0])
	}
//-------------------------------卖方
	accKeySell := settlements.SellUid + strconv.Itoa(ACCMOUNT)
	fmt.Println("i settlements.SellUid  : " + accKeySell)

	accBytes, _ := stub.GetState(accKeySell)
	accInfo1 := &AccountInfo{}
	errrr := json.Unmarshal(accBytes, accInfo1)
	if nil != errrr {
		return shim.Error("InvokeUsedElectric Incorrect json comvert: " + string(accBytes))
	}
	fmt.Println("i settlements.SellUid  : " + strconv.FormatInt(accInfo1.Amount,10))

	accInfo1.Amount=accInfo1.Amount+(settlements.Price*settlements.Eletric)

	fmt.Println("i settlements.SellUid  : " + strconv.FormatInt(accInfo1.Amount,10) )


	accInfo1.SumEs=accInfo1.SumEs+settlements.Eletric

	bytes, _ := json.Marshal(accInfo1)

	accErr := stub.PutState(accKeySell,bytes)
	if accErr != nil {
		return shim.Error("卖方---error:" + accErr.Error())
	}
//-------------------------------买方
	return SettlementsInvoke(stub,settlements)

}

func SettlementsInvoke(stub shim.ChaincodeStubInterface,settlements *Settlements) pb.Response {

	accKey := settlements.BuyUid+ strconv.Itoa(ACCMOUNT)
	fmt.Println("i settlements.buyUid  : " + accKey)

	accBytes, _ := stub.GetState(accKey)
	accInfo := &AccountInfo{}
	errrr := json.Unmarshal(accBytes, accInfo)
	if nil != errrr {
		return shim.Error("SettlementsInvoke   Incorrect json comvert: " + string(accBytes))
	}
	fmt.Println("i settlements.buyid  : " + strconv.FormatInt(accInfo.Amount,10) )

	accInfo.Amount = accInfo.Amount - settlements.Price*settlements.Eletric
	accInfo.SumEs = accInfo.SumEs + settlements.Eletric  //累加电量
	if accInfo.Amount < 0 {
		return shim.Error("用电方扣减金额大于余额: " + string(accBytes))
	}
	fmt.Println("i settlements.buyid  : " + strconv.FormatInt(accInfo.Amount,10) )


	usedElectric :=&UsedElectric{}
	usedElectric.Price=settlements.Price
	usedElectric.UID=settlements.BuyUid
	usedElectric.UsedElectric=settlements.Eletric
	bytes, _ := json.Marshal(accInfo)
	usedElectricBytes, _ := json.Marshal(usedElectric)
	usedElectricKey := usedElectric.UID + strconv.Itoa(USEDElECTRIC)

	if nil == stub.PutState(usedElectricKey, usedElectricBytes) &&
		nil == stub.PutState(accKey, bytes) {
		return shim.Success(nil)
	}
	return shim.Error("发电方交易保存失败：" + usedElectric.UID)

}


//用电方交易信息
func (usedElectric *UsedElectric) InvokeUsedElectric(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	fmt.Println("invokeUsedElectric args : " + args[0])

	errs := json.Unmarshal([]byte(args[0]), &usedElectric)
	if nil != errs {
		return shim.Error("initAcc args  Incorrect json comvert: " + args[0])
	}
	accKey := usedElectric.UID + strconv.Itoa(ACCMOUNT)
	accBytes, _ := stub.GetState(accKey)
	accInfo := &AccountInfo{}
	errrr := json.Unmarshal(accBytes, accInfo)
	if nil != errrr {
		return shim.Error("InvokeUsedElectric   Incorrect json comvert: " + string(accBytes))
	}
	accInfo.Amount = accInfo.Amount - usedElectric.Price*usedElectric.UsedElectric
	// accInfo.UsedAmount = accInfo.UsedAmount + usedElectric.Price*usedElectric.UsedElectric
	accInfo.SumEs = accInfo.SumEs + usedElectric.UsedElectric   //累加电量
	// accInfo.UsedEs = accInfo.UsedEs + usedElectric.UsedElectric //用电方累加电量和实际使用电量一致
	if accInfo.Amount < 0 {
		return shim.Error("用电方扣减金额大于余额: " + string(accBytes))
	}
	bytes, _ := json.Marshal(accInfo)
	usedElectricBytes, _ := json.Marshal(usedElectric)
	stub.PutState(accKey, bytes)

	usedElectricKey := usedElectric.UID + strconv.Itoa(USEDElECTRIC)

	if nil == stub.PutState(usedElectricKey, usedElectricBytes) &&
		nil == stub.PutState(accKey, bytes) {
		return shim.Success(nil)
	}
	return shim.Error("发电方交易保存失败：" + usedElectric.UID)
}

//获取单个用电方交易信息历史
func (usedElectric *UsedElectric) UsedElectricHis(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	fmt.Println("UsedElectricHis args : " + args[0])
	usedElectricKey := args[0] + strconv.Itoa(USEDElECTRIC)
	it, err := stub.GetHistoryForKey(usedElectricKey)
	if nil != err {
		return shim.Error(err.Error())
	}
	var result, _ = getHistoryListResult(it)
	return shim.Success(result)

}




//发电方报价信息：
type CreateElectric struct {
	User
	//发电量
	Electric int64 `json:"electric"`
	//电费报价
	Price int64 `json:"price"`

	//报价时间
	CreateTime string `json:"createTime"`
}

//发电方报价信息
func (createElectric *CreateElectric) InvokeCreateElectric(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	fmt.Println("InvokeCreateElectric args : " + args[0])

	errs := json.Unmarshal([]byte(args[0]), &createElectric)
	if nil != errs {
		return shim.Error("initAcc args  Incorrect json comvert: " + args[0])
	}
	accKey := createElectric.UID + strconv.Itoa(ACCMOUNT)
	accBytes, _ := stub.GetState(accKey)
	accInfo := &AccountInfo{}
	errrr := json.Unmarshal(accBytes, accInfo)
	if nil != errrr {
		return shim.Error("InvokeUsedElectric   Incorrect json comvert: " + string(accBytes))
	}

	//
	// accInfo.SumEs = accInfo.SumEs + createElectric.Electric //发电方报电总量
	//accInfo.Amount = accInfo.Amount + createElectric.Electric*createElectric.Price

	// bytes, _ := json.Marshal(accInfo)
	 createElectricBytes, _ := json.Marshal(createElectric)
	// stub.PutState(accKey, bytes)

	createElectricKey := createElectric.UID + strconv.Itoa(CREATELECTRIC)

	if nil == stub.PutState(createElectricKey, createElectricBytes){
	//  &&nil == stub.PutState(accKey, bytes) {
		return shim.Success(nil)
	}
	return shim.Error("发电方交易保存失败：" + createElectric.UID)
}

//获取单个发电方报价历史
func (createElectric *CreateElectric) CreateElectricHis(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	fmt.Println("CreateElectricHis args : " + args[0])
	createElectricKey := args[0] + strconv.Itoa(CREATELECTRIC)
	it, err := stub.GetHistoryForKey(createElectricKey)
	if nil != err {
		return shim.Error(err.Error())
	}
	var result, _ = getHistoryListResult(it)
	return shim.Success(result)

}

func getHistoryListResult(resultsIterator shim.HistoryQueryIteratorInterface) ([]byte, error) {

	defer resultsIterator.Close()
	fmt.Printf("queryResult")

	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
	
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		// item, _ := json.Marshal(queryResponse)
		buffer.WriteString(string(queryResponse.Value))
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")
	fmt.Printf("queryResult:\n%s\n", buffer.String())
	return buffer.Bytes(), nil
}

//业务
type Business struct {
}

func (business *Business) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("init......")
	return shim.Success(nil)
}

func (business *Business) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("Invoke")
	acc := &AccountInfo{}
	usedElectric := &UsedElectric{}
	createElectric := &CreateElectric{}
	settlements :=&Settlements{}
	function, args := stub.GetFunctionAndParameters()
	if function == "initAcc" { //初始化账户信息
		return acc.InitAcc(stub, args)
	} else if function == "queryAcc" { //查询账户信息
		return acc.QueryAcc(stub, args)
	} else if function == "invokeUsedElectric" { //设置用电方信息
		return usedElectric.InvokeUsedElectric(stub, args)
	} else if function == "usedElectricHis" {
		return usedElectric.UsedElectricHis(stub, args)
	} else if function == "invokeCreateElectric" {
		return createElectric.InvokeCreateElectric(stub, args)
	} else if function == "createElectricHis" {
		return createElectric.CreateElectricHis(stub, args)
	} else if function == "settlement" {
		return settlements.SettlementsElectric(stub, args)
	}

	return shim.Error("Invalid invoke function name. Expecting \"invoke\" \"delete\" \"query\"")
}

func main() {
	err := shim.Start(new(Business))
	if err != nil {
		fmt.Printf("Error starting Business chaincode: %s", err)
	}
}
