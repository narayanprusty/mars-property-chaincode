package main

import (
	"fmt"
	"encoding/json"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/msp"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/util"
	"crypto"
  "crypto/rsa"
  "crypto/x509"
  "encoding/base64"
	"encoding/pem"
)

type Property struct {
	Location	string `json:"location"`
	Owner string `json:"owner"`
	CreatedOn int64 `json:"createdOn"`
	History []string `json:"history"`
}

type User struct {
	PublicKey	string `json:"publicKey"`
	MetadataHash string `json:"metadataHash"`
	Permissions []string `json:"permissions"`
}

type PropertyChaincode struct {
}

func (t *PropertyChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	var err error
	var identity []byte

	identity, err = stub.GetCreator()
	
	if err != nil {
		return shim.Error("An error occured")
	}

	sId := &msp.SerializedIdentity{}
	err = proto.Unmarshal(identity, sId)
	
	if err != nil {
			return shim.Error("An error occured")
	}

	nodeId := sId.Mspid
	err = stub.PutState("propertyAuthority", []byte(nodeId))

	if err != nil {
		return shim.Error("An error occured")
	}

	return shim.Success(nil)
}

func (t *PropertyChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	if function == "getCreatorIdentity" {
		return t.getCreatorIdentity(stub, args)
	} else if function == "addProperty" {
		return t.addProperty(stub, args)
	} else if function == "getProperty" {
		return t.getProperty(stub, args)
	} else if function == "transferProperty" {
		return t.transferProperty(stub, args)
	}

	return shim.Error("Invalid function name: " + function)
}

func (t *PropertyChaincode) getCreatorIdentity(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	identity, err := stub.GetState("propertyAuthority")

	if err != nil {
		return shim.Error("An error occured")
	}

	if identity == nil {
		return shim.Error("Identity not yet stored")
	}

	return shim.Success(identity)
}

func (t *PropertyChaincode) addProperty(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments.")
	}

	var err error

	propertyAuthority, err := stub.GetState("propertyAuthority")

	if err != nil {
		return shim.Error("An error occured")
	}

	identity, err := stub.GetCreator()

	if err != nil {
		return shim.Error("An error occured")
	}

	sId := &msp.SerializedIdentity{}
	err = proto.Unmarshal(identity, sId)
	
	if err != nil {
			return shim.Error("An error occured")
	}

	nodeId := sId.Mspid

	if string(propertyAuthority) != nodeId {
		return shim.Error("You are not authorized")
	}

	propertyExists, err := stub.GetState("property_" + args[0])

	if propertyExists != nil  {
		return shim.Error("Property already exists")
	}

	var newProperty Property

	newProperty.Location = args[1]
	newProperty.Owner = args[2]

	currentTime, err := stub.GetTxTimestamp()
	newProperty.CreatedOn = currentTime.Seconds
	newProperty.History = append(newProperty.History, args[2])

	newPropertyJson, err := json.Marshal(newProperty)

	if err != nil {
			return shim.Error("An error occured")
	}

	err = stub.PutState("property_" + args[0], newPropertyJson)

	if err != nil {
		return shim.Error("An error occured")
	}

	return shim.Success(nil)
}

func (t *PropertyChaincode) getProperty(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments.")
	}
	
	property, err := stub.GetState("property_" + args[0])

	if err != nil {
		return shim.Error("An error occured")
	}

	return shim.Success(property)
}

func (t *PropertyChaincode) transferProperty(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments.")
	}

	var err error

	id := args[0]
	newOwner := args[1]

	signature, err := base64.StdEncoding.DecodeString(args[2])

	if err != nil {
		return shim.Error("An error occured")
	}

	identityChannelName := args[3]

	identity, err := stub.GetCreator()

	if err != nil {
		return shim.Error("An error occured")
	}

	sId := &msp.SerializedIdentity{}
	err = proto.Unmarshal(identity, sId)
	
	if err != nil {
		return shim.Error("An error occured")
	}

	propertyAuthority, err := stub.GetState("propertyAuthority")

	nodeId := sId.Mspid

	if string(propertyAuthority) != nodeId {
		return shim.Error("You are not authorized")
	}

	property, err := stub.GetState("property_" + id)
	var propertyStruct Property

	err = json.Unmarshal(property, &propertyStruct)

	if err != nil {
		return shim.Error("An error occured while unmarshal")
	}

	chainCodeArgs := util.ToChaincodeArgs("getIdentity", propertyStruct.Owner)
	response := stub.InvokeChaincode("identity", chainCodeArgs, identityChannelName)

	if response.Status != shim.OK {
		return shim.Error(response.Message)
 	}

	var userStruct User
	err = json.Unmarshal(response.Payload, &userStruct)

	if err != nil {
		return shim.Error("User struct creation failed")
	}

	userPublicKey, err := base64.StdEncoding.DecodeString(userStruct.PublicKey)

	block, _ := pem.Decode(userPublicKey)

	if block == nil {
    return shim.Error("Pem decoded")
	}
	
	userPublicKeyObj, err := x509.ParsePKIXPublicKey(block.Bytes)

	if err != nil {
		return shim.Error("Public key invalid")
	}

	message := []byte("{\"action\":\"transfer\",\"to\":\"" + newOwner + "\"}")

	newhash := crypto.SHA256
  pssh := newhash.New()
  pssh.Write(message)
	hashed := pssh.Sum(nil)
	
	rsaPublickey, _ := userPublicKeyObj.(*rsa.PublicKey)
	
	err = rsa.VerifyPKCS1v15(rsaPublickey, crypto.SHA256, hashed, signature)

	if err != nil {
		return shim.Error("Signature invalid")
	}

	propertyStruct.Owner = newOwner
	propertyStruct.History = append(propertyStruct.History, newOwner)

	propertyJson, err := json.Marshal(propertyStruct)

	err = stub.PutState("property_" + id, propertyJson)

	if err != nil {
		return shim.Error("An error occured while writing property")
	}
	
	return shim.Success(nil)
}

func main() {
	err := shim.Start(new(PropertyChaincode))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
