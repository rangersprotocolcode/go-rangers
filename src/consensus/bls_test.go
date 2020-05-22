package consensus

import (
	"testing"
	"x/src/consensus/base"
	"fmt"
	"github.com/stretchr/testify/assert"
	"encoding/hex"
	"math"
	"x/src/consensus/groupsig"
	"x/src/consensus/model"
	"math/rand"
	"bytes"
	"io/ioutil"
	"strings"
	"x/src/storage/sha3"
	"time"
)

//---------------------------------------Function Test-----------------------------------------------------------------
func TestKeyLength(test *testing.T) {
	secKey := groupsig.NewSeckeyFromRand(base.NewRand())
	publicKey := groupsig.GeneratePubkey(*secKey)

	fmt.Printf("secKey:%v,len:%d\n", secKey.Serialize(), len(secKey.Serialize()))
	fmt.Printf("pubkey :%v,len:%d\n", publicKey.Serialize(), len(publicKey.Serialize()))
	assert.Equal(test, len(secKey.Serialize()), 32)
	assert.Equal(test, len(publicKey.Serialize()), 128)

}

func TestSignAndVerifyOnce(test *testing.T) {
	runSignAndVerifyOnce(test)
}

func TestByFixKey(t *testing.T) {
	secKey := new(groupsig.Seckey)
	secKey.SetHexString("0x1c072ed882b56e42c34791586829d8f0805b306fb933c29bd727e9343a914596")
	pubkey := groupsig.GeneratePubkey(*secKey)
	message, _ := hex.DecodeString("524f1d03d1d81e94a099042736d40bd9681b867321443ff58a4568e274dbd83b")

	sig := groupsig.Sign(*secKey, message)
	verifyResult := groupsig.VerifySig(*pubkey, message, sig)
	assert.Equal(t, verifyResult, true)

	fmt.Printf("seckey:%v\n", secKey.GetHexString())
	fmt.Printf("pubkey:%v\n", pubkey.GetHexString())
	fmt.Printf("pubkey bytes:%v\n", pubkey.Serialize())
	fmt.Printf("message:%v\n", "0x"+hex.EncodeToString(message))
	fmt.Printf("sig:%v\n", sig.GetHexString())
}

func TestSignAndVerifyRepeatedly(test *testing.T) {
	var testCount = 1000
	for i := 0; i < testCount; i++ {
		runSignAndVerifyOnce(test)
	}
}

func genRandomKey() (secKey *groupsig.Seckey, publicKey *groupsig.Pubkey) {
	secKey = groupsig.NewSeckeyFromRand(base.NewRand())
	publicKey = groupsig.GeneratePubkey(*secKey)
	return
}

func genRandomMessage(length uint64) []byte {
	msg := make([]byte, length)

	var i uint64 = 0
	for ; i < length; i++ {
		msg[i] = byte(rand.Uint64() % 256)
	}
	return msg
}

func runSignAndVerifyOnce(test *testing.T) {
	secKey, publicKey := genRandomKey()
	msg := genRandomMessage(32)

	sign := groupsig.Sign(*secKey, msg)

	fmt.Printf("secKey:%v,len:%d\n", secKey.GetHexString(), len(secKey.GetHexString()))
	fmt.Printf("publicKey:%v,len:%d\n", publicKey.GetHexString(), len(publicKey.GetHexString()))
	fmt.Printf("msg:%v,len:%d\n", hex.EncodeToString(msg), len(publicKey.GetHexString()))

	fmt.Printf("sign:%v,len:%d\n", sign.GetHexString(), len(sign.Serialize()))
	verifyResult := groupsig.VerifySig(*publicKey, msg, sign)
	fmt.Printf("verifyResult:%v\n", verifyResult)
}

func TestMockGroupSign(test *testing.T) {
	groupSignAndVerify(17)
}

type testMinerInfo struct {
	SecretSeed     base.Rand
	MinerSeckey    groupsig.Seckey
	MinerPublicKey groupsig.Pubkey

	ReceivedSharePiece    []*model.SharePiece
	SignPrivateKeyInGroup *groupsig.Seckey

	ID groupsig.ID
}

func groupSignAndVerify(groupMemberNum uint64) {
	groupMemberList := createGroupMembers(groupMemberNum)
	threshold := int(math.Ceil(float64(groupMemberNum*51) / 100))

	mockGenSharePiece(threshold, groupMemberList)
	groupPublicKey := mockGotAllSharePiece(groupMemberList)
	fmt.Printf("Group publicKey:%s\n", groupPublicKey.GetHexString())

	message := genRandomMessage(32)
	fmt.Printf("message:%s\n", hex.EncodeToString(message))
	memberSignMap := make(map[string]groupsig.Signature, 0)

	fmt.Printf("Begin recover.threshold:%d\n", threshold)
	for _, member := range groupMemberList {
		sign := groupsig.Sign(*member.SignPrivateKeyInGroup, message)
		memberSignMap[member.ID.GetHexString()] = sign
		fmt.Printf("ID:%s,Sign:%s\n", member.ID.GetHexString(), sign.GetHexString())
	}
	groupSign := groupsig.RecoverGroupSignature(memberSignMap, threshold)
	fmt.Printf("Group sign:%s\n", groupSign.GetHexString())

	verifyResult := groupsig.VerifySig(groupPublicKey, message, *groupSign)
	if !verifyResult {
		panic("Group sign verify failed! Please contact the developer.")
	}
}

func createGroupMembers(groupMemberNum uint64) []*testMinerInfo {
	groupMemberList := make([]*testMinerInfo, 0)
	var i uint64 = 0
	for ; i < groupMemberNum; i++ {
		miner := testMinerInfo{}
		miner.SecretSeed = base.NewRand()
		miner.MinerSeckey = *groupsig.NewSeckeyFromRand(miner.SecretSeed)
		miner.MinerPublicKey = *groupsig.GeneratePubkey(miner.MinerSeckey)
		miner.ReceivedSharePiece = make([]*model.SharePiece, 0)
		miner.ID.Deserialize(miner.SecretSeed.Bytes())

		groupMemberList = append(groupMemberList, &miner)
	}
	return groupMemberList
}

func mockGenSharePiece(threshold int, groupMemberList []*testMinerInfo) {
	for i := 0; i < len(groupMemberList); i++ {
		miner := groupMemberList[i]
		genSharePiece(threshold, *miner, groupMemberList)
	}
}

func genSharePiece(threshold int, minerInfo testMinerInfo, groupMemberList []*testMinerInfo) {
	fmt.Printf("Begin generate share piece.Member id:%s\n", minerInfo.ID.GetHexString())
	secretList := make([]groupsig.Seckey, threshold)
	for i := 0; i < threshold; i++ {
		secretList[i] = *groupsig.NewSeckeyFromRand(minerInfo.SecretSeed.Deri(i))
	}
	fmt.Printf("Seckey list:\n")
	for _, seckey := range secretList {
		fmt.Printf("%s\n", seckey.GetHexString())
	}

	seedSecKey := groupsig.NewSeckeyFromRand(minerInfo.SecretSeed.Deri(0))
	seedPubkey := groupsig.GeneratePubkey(*seedSecKey)
	fmt.Printf("seed seckey:%v,seed seckey bytes:%v:\n", seedSecKey.GetHexString(), seedSecKey.Serialize())
	fmt.Printf("seed pubkey:%v:\n", seedPubkey.GetHexString())

	for i := 0; i < len(groupMemberList); i++ {
		miner := groupMemberList[i]
		sharePiece := new(model.SharePiece)
		sharePiece.Pub = *seedPubkey
		sharePiece.Share = *groupsig.ShareSeckey(secretList, miner.ID)
		fmt.Printf("Generate share piece.Target id:%s,piece:%s\n", miner.ID.GetHexString(), sharePiece.Share.GetHexString())

		miner.ReceivedSharePiece = append(miner.ReceivedSharePiece, sharePiece)
	}
	fmt.Printf("\n")
}

func mockGotAllSharePiece(groupMemberList []*testMinerInfo) groupsig.Pubkey {
	signPublicKeyList := make([]groupsig.Pubkey, 0)
	fmt.Printf("Aggregate received share piece.\n")
	for index, member := range groupMemberList {
		fmt.Printf("Member id:%s.\n", member.ID.GetHexString())
		receivedShareList := make([]groupsig.Seckey, 0)
		for _, sharePiece := range member.ReceivedSharePiece {
			receivedShareList = append(receivedShareList, sharePiece.Share)
			fmt.Printf("Rceceived share piece:%s.\n", sharePiece.Share.GetHexString())
			if index == 0 {
				signPublicKeyList = append(signPublicKeyList, sharePiece.Pub)
				fmt.Printf("Rceceived pubkey:%s.\n", sharePiece.Pub.GetHexString())
			}
		}
		signPrivateKeyInGroup := groupsig.AggregateSeckeys(receivedShareList)
		fmt.Printf("sign private key in group:%s\n\n", signPrivateKeyInGroup.GetHexString())
		groupMemberList[index].SignPrivateKeyInGroup = signPrivateKeyInGroup
	}
	groupPublicKey := groupsig.AggregatePubkeys(signPublicKeyList)
	return *groupPublicKey
}

//---------------------------------------Benchmark Test-----------------------------------------------------------------
var testCount = 100
var privateKeyList = make([]groupsig.Seckey, testCount)
var publicKeyList = make([]groupsig.Pubkey, testCount)
var messageList = make([][]byte, testCount)
var signList = make([]groupsig.Signature, testCount)

func BenchmarkSign(b *testing.B) {
	prepareData()
	b.ResetTimer()

	begin := time.Now()
	for i := 0; i < testCount; i++ {
		secKey := privateKeyList[i]
		message := messageList[i]
		signList[i] = groupsig.Sign(secKey, message)
	}
	fmt.Printf("cost:%v\n", time.Since(begin).Seconds())
}

func BenchmarkVerify(b *testing.B) {
	prepareData()
	b.ResetTimer()

	for i := 0; i < testCount; i++ {
		publicKey := publicKeyList[i]
		message := messageList[i]
		sign := signList[i]
		verifyResult := groupsig.VerifySig(publicKey, message, sign)
		if !verifyResult {
			panic("Verify sign failed!")
		}
	}
}

func BenchmarkSignAndVerify(b *testing.B) {
	prepareData()
	b.ResetTimer()

	for i := 0; i < testCount; i++ {
		secKey := privateKeyList[i]
		publicKey := publicKeyList[i]
		message := messageList[i]
		sign := groupsig.Sign(secKey, message)
		verifyResult := groupsig.VerifySig(publicKey, message, sign)
		if !verifyResult {
			panic("Verify sign failed!")
		}
	}
}

func prepareData() {
	for i := 0; i < testCount; i++ {
		var secKey *groupsig.Seckey
		var publicKey *groupsig.Pubkey
		secKey, publicKey = genRandomKey()

		privateKeyList[i] = *secKey
		publicKeyList[i] = *publicKey
		messageList[i] = genRandomMessage(32)
		signList[i] = groupsig.Sign(*secKey, messageList[i])
	}
}

//---------------------------------------Standard Data Test---------------------------------------------------------------
const prefix = "0x"

func TestGenComparisonData(test *testing.T) {
	fileName := "bls_comparisonData_go.txt"

	var buffer bytes.Buffer
	for i := 0; i < testCount; i++ {
		var groupMemberNum uint64 = 3
		groupMemberList := make([]*testMinerInfo, 0)
		var j uint64 = 0
		for ; j < groupMemberNum; j++ {
			miner := testMinerInfo{}
			miner.ID.Deserialize(genRandomMessage(32))
			miner.SecretSeed = base.RandFromBytes(miner.ID.Serialize())
			miner.MinerSeckey = *groupsig.NewSeckeyFromRand(miner.SecretSeed)
			miner.MinerPublicKey = *groupsig.GeneratePubkey(miner.MinerSeckey)
			miner.ReceivedSharePiece = make([]*model.SharePiece, 0)

			groupMemberList = append(groupMemberList, &miner)
		}

		threshold := int(math.Ceil(float64(groupMemberNum*51) / 100))
		buffer.WriteString("idList:")
		for index, member := range groupMemberList {
			buffer.WriteString(member.ID.GetHexString())
			if index < len(groupMemberList)-1 {
				buffer.WriteString("&")
			}
		}

		mockGenSharePiece(threshold, groupMemberList)
		groupPublicKey := mockGotAllSharePiece(groupMemberList)
		buffer.WriteString("|groupPublicKey:")
		buffer.WriteString(groupPublicKey.GetHexString())

		messageByte := genRandomMessage(32)
		buffer.WriteString("|message:")
		message := prefix + hex.EncodeToString(messageByte)
		buffer.WriteString(message)

		memberSignMap := make(map[string]groupsig.Signature, 0)
		for _, member := range groupMemberList {
			sign := groupsig.Sign(*member.SignPrivateKeyInGroup, messageByte)
			memberSignMap[member.ID.GetHexString()] = sign
		}
		groupSign := groupsig.RecoverGroupSignature(memberSignMap, threshold)
		buffer.WriteString("|groupSign:")
		buffer.WriteString(groupSign.GetHexString())
		if i < testCount-1 {
			buffer.WriteString("\n")
		}
	}
	ioutil.WriteFile(fileName, buffer.Bytes(), 0644)
}

func TestValidateComparisonData(test *testing.T) {
	fileName := "bls_comparisonData_java.txt"

	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic("read bls_comparisonData_java info  file error:" + err.Error())
	}
	records := strings.Split(string(bytes), "\n")

	for _, record := range records {
		//fmt.Println(record)
		elements := strings.Split(record, "|")
		//fmt.Println(elements[0])
		//fmt.Println(elements[1])
		//fmt.Println(elements[2])
		//fmt.Println(elements[3])

		idListStr := strings.Replace(elements[0], "idList:", "", 1)
		idList := strings.Split(idListStr, "&")
		groupPublicKey := strings.Replace(elements[1], "groupPublicKey:", "", 1)
		message := strings.Replace(elements[2], "message:", "", 1)
		groupSign := strings.Replace(elements[3], "groupSign:", "", 1)

		//fmt.Println(idList)
		//fmt.Println(groupPublicKey)
		fmt.Println(message)
		//fmt.Println(groupSign)
		validateFunction(idList, groupPublicKey, message, groupSign, test)
	}
}

func validateFunction(idList []string, groupPublicKey, messageStr, groupSign string, test *testing.T) {
	//validate groupPublicKey
	groupMemberList := make([]*testMinerInfo, 0)
	for _, idStr := range idList {
		miner := testMinerInfo{}
		miner.ID.SetHexString(idStr)
		miner.SecretSeed = base.RandFromBytes(miner.ID.Serialize())
		miner.MinerSeckey = *groupsig.NewSeckeyFromRand(miner.SecretSeed)
		miner.MinerPublicKey = *groupsig.GeneratePubkey(miner.MinerSeckey)
		miner.ReceivedSharePiece = make([]*model.SharePiece, 0)

		fmt.Printf("minerInfo. id:%s,secretSeed:%s,seckey:%s,publickey:%s\n", miner.ID.GetHexString(), miner.SecretSeed.GetHexString(), miner.MinerSeckey.GetHexString(), miner.MinerPublicKey.GetHexString())
		groupMemberList = append(groupMemberList, &miner)
	}
	threshold := int(math.Ceil(float64(3*51) / 100))

	mockGenSharePiece(threshold, groupMemberList)
	exceptedGroupPublicKey := mockGotAllSharePiece(groupMemberList)
	//fmt.Printf("Group publicKey:%s\n", exceptedGroupPublicKey.GetHexString())
	assert.Equal(test, exceptedGroupPublicKey.GetHexString(), groupPublicKey)

	//validate groupSign
	message, _ := hex.DecodeString(messageStr[2:])
	memberSignMap := make(map[string]groupsig.Signature, 0)
	for _, member := range groupMemberList {
		sign := groupsig.Sign(*member.SignPrivateKeyInGroup, message)
		memberSignMap[member.ID.GetHexString()] = sign
		fmt.Printf("Id:%s,sekey:%s,message:%s,sign:%s\n", member.ID.GetHexString(), member.MinerSeckey.GetHexString(), hex.EncodeToString(message), sign.GetHexString())
	}
	exceptedGroupSign := groupsig.RecoverGroupSignature(memberSignMap, threshold)
	//fmt.Printf("Group sign:%s\n", exceptedGroupSign.GetHexString())
	assert.Equal(test, exceptedGroupSign.GetHexString(), groupSign)

	//verify group sign
	verifyResult := groupsig.VerifySig(exceptedGroupPublicKey, message, *exceptedGroupSign)
	if !verifyResult {
		panic("Group sign verify failed! Please contact the developer.")
	}

}

func TestSha3(t *testing.T) {
	a := []byte{1, 2, 3}
	sha := sha3.New256()
	sha.Write(a)

	m := new([]byte)
	r := sha.Sum(*m)
	fmt.Printf("r:%v\n", r)
}

func TestSign(t *testing.T) {
	secKeyBytes, _ := hex.DecodeString("36b71345409af1ea864e4a8bcde2bbb8c6cc5e778244ef3a8703ded84737309f")
	message, _ := hex.DecodeString("c5780c2ba9d0311bdf6e227bea3306b7c5cef776bd04f934d3534407f95ef671")

	secKey := new(groupsig.Seckey)
	secKey.Deserialize(secKeyBytes)
	sign := groupsig.Sign(*secKey, message)
	fmt.Printf("sign:%s\n", sign.GetHexString())
}
