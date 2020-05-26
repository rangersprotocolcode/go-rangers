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
	"time"
	"x/src/common"
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
	for i := 0; i < 1000; i++ {
		groupSignAndVerify(10)
	}
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
	//fmt.Printf("Begin generate share piece.Member id:%s\n", minerInfo.ID.GetHexString())
	secretList := make([]groupsig.Seckey, threshold)
	for i := 0; i < threshold; i++ {
		secretList[i] = *groupsig.NewSeckeyFromRand(minerInfo.SecretSeed.Deri(i))
	}
	//fmt.Printf("Seckey list:\n")
	//for _, seckey := range secretList {
	//	fmt.Printf("%s\n", seckey.GetHexString())
	//}

	seedSecKey := groupsig.NewSeckeyFromRand(minerInfo.SecretSeed.Deri(0))
	seedPubkey := groupsig.GeneratePubkey(*seedSecKey)
	//fmt.Printf("seed seckey:%v,\n", seedSecKey.GetHexString())
	//fmt.Printf("seed pubkey:%v:\n", seedPubkey.GetHexString())

	for i := 0; i < len(groupMemberList); i++ {
		miner := groupMemberList[i]
		sharePiece := new(model.SharePiece)
		sharePiece.Pub = *seedPubkey
		sharePiece.Share = *groupsig.ShareSeckey(secretList, miner.ID)
		//fmt.Printf("Generate share piece.Target id:%s,piece:%s\n", miner.ID.GetHexString(), sharePiece.Share.GetHexString())

		miner.ReceivedSharePiece = append(miner.ReceivedSharePiece, sharePiece)
	}
	//fmt.Printf("\n")
}

func mockGotAllSharePiece(groupMemberList []*testMinerInfo) groupsig.Pubkey {
	signPublicKeyList := make([]groupsig.Pubkey, 0)
	//fmt.Printf("Aggregate received share piece.\n")
	for index, member := range groupMemberList {
		//fmt.Printf("Member id:%s.\n", member.ID.GetHexString())
		receivedShareList := make([]groupsig.Seckey, 0)
		for _, sharePiece := range member.ReceivedSharePiece {
			receivedShareList = append(receivedShareList, sharePiece.Share)
			//fmt.Printf("Rceceived share piece:%s.\n", sharePiece.Share.GetHexString())
			if index == 1 {
				signPublicKeyList = append(signPublicKeyList, sharePiece.Pub)
				//fmt.Printf("Rceceived pubkey:%s.\n", sharePiece.Pub.GetHexString())
			}
		}
		signPrivateKeyInGroup := groupsig.AggregateSeckeys(receivedShareList)
		//fmt.Printf("sign private key in group:%s\n\n", signPrivateKeyInGroup.GetHexString())
		groupMemberList[index].SignPrivateKeyInGroup = signPrivateKeyInGroup
	}

	for _, pubkey := range signPublicKeyList {
		fmt.Printf("Rceceived pubkey:%s.\n", pubkey.GetHexString())
	}

	pubkeyList := getPubkeyList1()
	for _, pubkey := range pubkeyList {
		fmt.Printf("Got pubkey:%v.\n", pubkey.Serialize())
	}

	groupPublicKey := groupsig.AggregatePubkeys(signPublicKeyList)
	fmt.Printf("Group pubkey:%s\n", groupPublicKey.GetHexString())

	groupPubkey := groupsig.AggregatePubkeys(pubkeyList)
	fmt.Printf("Group pubkey:%s\n", groupPubkey.GetHexString())
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

	begin := time.Now()
	for i := 0; i < testCount; i++ {
		publicKey := publicKeyList[i]
		message := messageList[i]
		sign := signList[i]
		verifyResult := groupsig.VerifySig(publicKey, message, sign)
		if !verifyResult {
			panic("Verify sign failed!")
		}
	}
	fmt.Printf("cost:%v\n", time.Since(begin).Seconds())
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

//----------------------------------------------------------------------------------------------------------------

func TestMockCreateGroupByFixID(t *testing.T) {
	idList := getIdList()
	groupMemberList := createGroupMemberByFixID(idList)
	threshold := int(math.Ceil(float64(len(idList)*51) / 100))

	mockGenSharePiece(threshold, groupMemberList)
	groupPublicKey := mockGotAllSharePiece(groupMemberList)
	//fmt.Printf("Group publicKey:%s\n", groupPublicKey.GetHexString())

	message := genRandomMessage(32)
	//fmt.Printf("message:%s\n", hex.EncodeToString(message))
	memberSignMap := make(map[string]groupsig.Signature, 0)

	//fmt.Printf("Begin recover.threshold:%d\n", threshold)
	for _, member := range groupMemberList {
		sign := groupsig.Sign(*member.SignPrivateKeyInGroup, message)
		memberSignMap[member.ID.GetHexString()] = sign
		//fmt.Printf("ID:%s,Sign:%s\n", member.ID.GetHexString(), sign.GetHexString())
	}
	groupSign := groupsig.RecoverGroupSignature(memberSignMap, threshold)
	//fmt.Printf("Group sign:%s\n", groupSign.GetHexString())

	verifyResult := groupsig.VerifySig(groupPublicKey, message, *groupSign)
	if !verifyResult {
		panic("Group sign verify failed! Please contact the developer.")
	}
}

func getIdList() []string {
	idList := make([]string, 0)
	idList = append(idList, "0xd2754a3ca343a50d279dd69325ef768734c2ecfcb7a3faef6ec23516680d90b7")
	idList = append(idList, "0x4b7cc1b4eb7aaab669f1a5eb9ebd382cfa6e2a28ac2f1ab6fbb0247219a00ad6")
	idList = append(idList, "0xedcf4a3457361ed75e9d4531362b5d1c8a482c94db3fdb815f821fc18ef59d89")
	idList = append(idList, "0xf8b871735a91ee0289c63f13cec425162d2edf75f22ae393934e99936a05f436")
	idList = append(idList, "0x8510cb890b11424b8d0986f7f3574af96fb00e452c0dbf658048a53a6451855d")
	idList = append(idList, "0x05b231ac9481c219957ca7965efecd24c84b2b2bdff6f4b18f090e5059aa87ea")
	idList = append(idList, "0x286595ade352c80fdf26b5863d4aed7ece85cffb0164d024b96e64fa0ae10a6d")
	idList = append(idList, "0x34f9079ff55e258ae82118acac676d7bbb014e2584875b3eebd5cb320c5613e2")
	idList = append(idList, "0xdd8d185f2f9c1fb8a990f790d344922e815497e2cb4924a1643c108add7f2823")
	idList = append(idList, "0x470bc19cb7d7a6f825fc8f9c167e55fc9c34eb260bf41e11d703889458e5ce61")
	idList = append(idList, "0x8946c872c1fe68164d44ce02a614ae0c18cbac44a2351cf82b5a551da8b53bf5")
	idList = append(idList, "0xf0ba4c2a694fb9906ec6d4d7e46f8feb9db5b6aa33c39bc39b11170c38123368")
	idList = append(idList, "0x97cd69b5a9f4d557574dcd1375f6d894fd130906cea2af42bdb0888908e07a44")
	idList = append(idList, "0xa090959f97c3bd5fafedfe7b4a34db963c2421b29e0586d339f22f0551ca2f6e")
	idList = append(idList, "0x57457f5427671af40984765692d8e0f1c5c9a0ed79d3e9828ef39af7b81b71ea")

	return idList
}

func createGroupMemberByFixID(groupMemberIDList []string) []*testMinerInfo {
	groupMemberList := make([]*testMinerInfo, 0)

	h, _ := hex.DecodeString("bce7034201a3a9b73978abc5a2f4dc72b1932110dcec474ad79ec5f39f658ce3")
	r := base.RandFromBytes(h)

	for _, id := range groupMemberIDList {
		miner := testMinerInfo{}

		miner.SecretSeed = base.RandFromBytes(common.FromHex(id)).DerivedRand(r[:])
		miner.MinerSeckey = *groupsig.NewSeckeyFromRand(miner.SecretSeed)
		miner.MinerPublicKey = *groupsig.GeneratePubkey(miner.MinerSeckey)
		miner.ReceivedSharePiece = make([]*model.SharePiece, 0)
		miner.ID.Deserialize(common.FromHex(id))
		groupMemberList = append(groupMemberList, &miner)
	}
	return groupMemberList
}

func TestAggregatePubkey(t *testing.T) {
	//pubkeyList := getPubkeyList1()
	//groupPubkey := groupsig.AggregatePubkeys(pubkeyList)
	//fmt.Printf("Group pubkey:%s\n",groupPubkey.GetHexString())

	p15 := new(groupsig.Pubkey)
	b15, _ := hex.DecodeString("00aba6bb62fd346ca87cbf201d686b83f6831fc14c5d2f54e894850d717b00e766d7e43e4adfe2c2f303394d1d93e2fa84b3d7534a7021ee53ef4abe29832fa083d445bdea0b684642dce08733a6190bc9d5129a0f2a75f5aa4a5a9e261d64e569a292b0c5c1377aacf1d226633b7a8c8308fcc44c2c9f31b1dcbfa29677f02e")
	fmt.Printf("bytes:%v\n", b15)
	p15.Deserialize(b15)

	b1 := p15.Serialize()
	fmt.Printf("bytes:%v\n", b1)
}

//func getPubkeyList() []groupsig.Pubkey {
//	pubkeyList := make([]groupsig.Pubkey, 0)
//
//	p1 := new(groupsig.Pubkey)
//	b1,_:= hex.DecodeString("47d55ab5ec48a754a8dbdf6cc5931308fea101d2350ad99a8264eeb74154af48704753262a1be5b7b91a644616e7415dd3b224889184c6752280a6de29bb277458dbc7f049b278d783ac622cdd7bd0699811e7b5e5195e5f9ac1dc86c9ba70086e29d57a0a4d4c02aab541458d1112f81b38a532a8076ffcd8a16e1efbe974e4")
//	p1.Deserialize(b1)
//	pubkeyList = append(pubkeyList,*p1)
//
//	p2 := new(groupsig.Pubkey)
//	b2,_:= hex.DecodeString("8e6bb4bc55d661ad37c5c78b5e508f2a7cb9606f42a039e5cf8a141a2dc1e818400ae7a849061bd2cb7388d5570f7aa111ad7a99d0c616bad08765e9376adacc8815b3accd14bb631a2512e18ac47d19d6eab6938510066726cb5368a61e944746f76320119bc9816df6953b03e160fccee6cdadf44346f7aa5dd835019b266b")
//	p2.Deserialize(b2)
//	pubkeyList = append(pubkeyList,*p2)
//
//	p3 := new(groupsig.Pubkey)
//	b3,_:= hex.DecodeString("12197c05e968d770c17cd58d2c47618d8c9d88f58a8baadba113d38355ee74691c8c4b356c6963df757612741b18d40bbe1e8694a97fa9add2525e46c307ed9530399b31e99001c6666d5cffbeb9536bddffbdef6b18329a2b89f23853e2f68b1e927ae9d725e0300fa391a0c3c69d15785fde809be0974ab9a523ecf9c8a5b4")
//	p3.Deserialize(b3)
//	pubkeyList = append(pubkeyList,*p3)
//
//	p4 := new(groupsig.Pubkey)
//	b4,_:= hex.DecodeString("0e7197971858b9ec6789ac21a779dce3bee34151c918d931f18b2be84773ee0c141d1d20415d10bb6b55910ee0271311b4b1f33274c47fab0966c3606321f7153b8d58d883fac1483e2ab739a878a0ede52126e1b14dde507c07ecdfcc631d0f67646b715a09036081a7c310a9a3cdd0b8d3de4b5dbc2bba4f202cfb213f15a6")
//	p4.Deserialize(b4)
//	pubkeyList = append(pubkeyList,*p4)
//
//	p5 := new(groupsig.Pubkey)
//	b5,_:= hex.DecodeString("3c1883eabb4833c2fd29e88030c644dc7d982f50370b820492dc1963c8a1e7f80123c7c040de5f587109fbe744f5563608d682374bfe29880d48f1cc942a3df800a7f1d84a3dd1f57f581794ace475c4b124516724e7500afa8f4ad28e4bd1b2029d98107281686b51d04a9e2fb9ee00812e47bda20f44686b8f10cdcee95d1d")
//	p5.Deserialize(b5)
//	pubkeyList = append(pubkeyList,*p5)
//
//	p6 := new(groupsig.Pubkey)
//	b6,_:= hex.DecodeString("862bf075d1ab229f16f4c13b56bbf193bd86ac9b1a1b1ffabea1a528a0dfd0e3339b0c74e816674f613fd87a3511322e4583940e36780e466d7f7e0a3ba63f132f7ab281b6c2dde5914e2524ae3258329c6bc01aa94e48883a668c9dc509aeea895d434f3b5dc255d361e3fc61165e492315c626c0c2a96636052a8e58c44f47")
//	p6.Deserialize(b6)
//	pubkeyList = append(pubkeyList,*p6)
//
//	p7 := new(groupsig.Pubkey)
//	b7,_:= hex.DecodeString("6004c2bf80ab851002d6b0fa36813d0af1c3f7566fd828736a92536ad3f7cb1a8f40a4b620ce0e20ba12e4bf589b5e2ca36080938b085a8857b8f0e3907c37b2708fa3a2ccc0501510dff11df85747d14e139f5ce64896f082e855dae71db6d338f79e656feac8afd30166230712aa61ccb6245dc2ee8ad018d27e6e8cb06128")
//	p7.Deserialize(b7)
//	pubkeyList = append(pubkeyList,*p7)
//
//	p8 := new(groupsig.Pubkey)
//	b8,_:= hex.DecodeString("89d62d291bec395de1c2a8459d85fa56797d5ea26a583d1129d985e7349f1b2a3e6d58167d1e6cd1d0c08d9eccc4b6e3100851a096293d795a137b506d782d4c3d709d2a954b8cc0b49392be3bd279ae788860a087c39fca5213c2b1426b9ffc6d0e9b06bf081a7a3b5456d15677ce8800b5f571a4704da6c6c38bd5973f5b6b")
//	p8.Deserialize(b8)
//	pubkeyList = append(pubkeyList,*p8)
//
//	p9 := new(groupsig.Pubkey)
//	b9,_:= hex.DecodeString("5a1a71596a3986792d52a35a7dfc592679f5b86b89c72cca6d8339373539d7e777d885eac730d03c32b6ee921aa1d72f30ac855248a57f2cb4db22e9771e0d21436a8cdfe2c4ff75e023ab9062b3ac1679e0e93afe24a3fce6d236cf309248954e43cb47d425aa9769f8f70849a9a6642c5e1ef2376b5631ad9917c341c5fbca")
//	p9.Deserialize(b9)
//	pubkeyList = append(pubkeyList,*p9)
//
//	p10 := new(groupsig.Pubkey)
//	b10,_:= hex.DecodeString("8829b0479aa60da699a2a3ac8d8ad382ee3fd12cbe1e83dcdfbf07a7a644bc1e2a08f3d9c0f997c719d998810dd5a5662bf599e69641f0a0efd47ab5c387706064cfe9ef814693d8f09f2f94a6f5dd5c11072e0715a4d2bf9d108429353626f974af71e947a128efad2ce2eedfff0f0498df01ac24368e7dfaeca5649ae58771")
//	p10.Deserialize(b10)
//	pubkeyList = append(pubkeyList,*p10)
//
//	p11 := new(groupsig.Pubkey)
//	b11,_:= hex.DecodeString("53136473e51089c5fb9bdfa3096d59e9e6c42d7fb417a3f838c2075df73f494e6c463966ec7c64b1fcb1713de6310e551ca1ed96216e6b4e1e11800f1a1f7c5288a6213fe5b1b98e9fdeca7a00d7f88d648b01d482c4ffdcfa01bac8971a97306f257b1b7b5bf5cb0e92626f8172a6a4340ec11856cdebfc3bb6c7a96777f096")
//	p11.Deserialize(b11)
//	pubkeyList = append(pubkeyList,*p11)
//
//	p12 := new(groupsig.Pubkey)
//	b12,_:= hex.DecodeString("72f63f0714b5d46f351d1a19968dfb862d497c235d04fcd9b0c8cd69a45781b35c717d2c61ef755c7e94d24017b7df871df2d06687685d17849c375ffa94cb467fc932516f790e5c614bee41237ddf4e3f57e04315afe9bcb66758a3e62ac1213161813b2bc7d83cf3f3825b12d53678b91e3a57b63c28380ad58ecb310e027a")
//	p12.Deserialize(b12)
//	pubkeyList = append(pubkeyList,*p12)
//
//	p13 := new(groupsig.Pubkey)
//	b13,_:= hex.DecodeString("1a20140d07c7a5e634140e489750603865363c2f0386707c3437e172ce587b5b33b2978329f54f49ea37f2717cfaf70af5b42c16cb8f0af93eef1567564d78b22f87bc3b0f20799d1e4a067240605b317019e7a152c7370a579ae99cbdd2f04b5124124bef0a13b0fd0ca37f908367ddb6500b6f02601ce06644d394a15cb917")
//	p13.Deserialize(b13)
//	pubkeyList = append(pubkeyList,*p13)
//
//	p14 := new(groupsig.Pubkey)
//	b14,_:= hex.DecodeString("3c5d1dba47c495ce10a658e9e5c67151835eced353c0024dcdfe1f925fc92a760258ddb993dda493a8d33a60152337067dccdafb4e0f2f9eaea46490da091923589813549dcba58160514b0c0b0c1a1d920d431c2152764891d976639e1983f45849fa357006e38b8f41c19218a09adfe23940528b674e05f7e11e7a8078bd66")
//	p14.Deserialize(b14)
//	pubkeyList = append(pubkeyList,*p14)
//
//	p15 := new(groupsig.Pubkey)
//	b15,_:= hex.DecodeString("00aba6bb62fd346ca87cbf201d686b83f6831fc14c5d2f54e894850d717b00e766d7e43e4adfe2c2f303394d1d93e2fa84b3d7534a7021ee53ef4abe29832fa083d445bdea0b684642dce08733a6190bc9d5129a0f2a75f5aa4a5a9e261d64e569a292b0c5c1377aacf1d226633b7a8c8308fcc44c2c9f31b1dcbfa29677f02e")
//	p15.Deserialize(b15)
//	pubkeyList = append(pubkeyList,*p15)
//	return pubkeyList
//}

func getPubkeyList1() []groupsig.Pubkey {
	pubkeyList := make([]groupsig.Pubkey, 0)

	p2 := new(groupsig.Pubkey)
	b2, _ := hex.DecodeString("8e6bb4bc55d661ad37c5c78b5e508f2a7cb9606f42a039e5cf8a141a2dc1e818400ae7a849061bd2cb7388d5570f7aa111ad7a99d0c616bad08765e9376adacc8815b3accd14bb631a2512e18ac47d19d6eab6938510066726cb5368a61e944746f76320119bc9816df6953b03e160fccee6cdadf44346f7aa5dd835019b266b")
	p2.Deserialize(b2)
	pubkeyList = append(pubkeyList, *p2)

	p11 := new(groupsig.Pubkey)
	b11, _ := hex.DecodeString("53136473e51089c5fb9bdfa3096d59e9e6c42d7fb417a3f838c2075df73f494e6c463966ec7c64b1fcb1713de6310e551ca1ed96216e6b4e1e11800f1a1f7c5288a6213fe5b1b98e9fdeca7a00d7f88d648b01d482c4ffdcfa01bac8971a97306f257b1b7b5bf5cb0e92626f8172a6a4340ec11856cdebfc3bb6c7a96777f096")
	p11.Deserialize(b11)
	pubkeyList = append(pubkeyList, *p11)

	p15 := new(groupsig.Pubkey)
	b15, _ := hex.DecodeString("00aba6bb62fd346ca87cbf201d686b83f6831fc14c5d2f54e894850d717b00e766d7e43e4adfe2c2f303394d1d93e2fa84b3d7534a7021ee53ef4abe29832fa083d445bdea0b684642dce08733a6190bc9d5129a0f2a75f5aa4a5a9e261d64e569a292b0c5c1377aacf1d226633b7a8c8308fcc44c2c9f31b1dcbfa29677f02e")
	p15.Deserialize(b15)
	pubkeyList = append(pubkeyList, *p15)

	p5 := new(groupsig.Pubkey)
	b5, _ := hex.DecodeString("3c1883eabb4833c2fd29e88030c644dc7d982f50370b820492dc1963c8a1e7f80123c7c040de5f587109fbe744f5563608d682374bfe29880d48f1cc942a3df800a7f1d84a3dd1f57f581794ace475c4b124516724e7500afa8f4ad28e4bd1b2029d98107281686b51d04a9e2fb9ee00812e47bda20f44686b8f10cdcee95d1d")
	p5.Deserialize(b5)
	pubkeyList = append(pubkeyList, *p5)

	p12 := new(groupsig.Pubkey)
	b12, _ := hex.DecodeString("72f63f0714b5d46f351d1a19968dfb862d497c235d04fcd9b0c8cd69a45781b35c717d2c61ef755c7e94d24017b7df871df2d06687685d17849c375ffa94cb467fc932516f790e5c614bee41237ddf4e3f57e04315afe9bcb66758a3e62ac1213161813b2bc7d83cf3f3825b12d53678b91e3a57b63c28380ad58ecb310e027a")
	p12.Deserialize(b12)
	pubkeyList = append(pubkeyList, *p12)

	p3 := new(groupsig.Pubkey)
	b3, _ := hex.DecodeString("12197c05e968d770c17cd58d2c47618d8c9d88f58a8baadba113d38355ee74691c8c4b356c6963df757612741b18d40bbe1e8694a97fa9add2525e46c307ed9530399b31e99001c6666d5cffbeb9536bddffbdef6b18329a2b89f23853e2f68b1e927ae9d725e0300fa391a0c3c69d15785fde809be0974ab9a523ecf9c8a5b4")
	p3.Deserialize(b3)
	pubkeyList = append(pubkeyList, *p3)

	p1 := new(groupsig.Pubkey)
	b1, _ := hex.DecodeString("47d55ab5ec48a754a8dbdf6cc5931308fea101d2350ad99a8264eeb74154af48704753262a1be5b7b91a644616e7415dd3b224889184c6752280a6de29bb277458dbc7f049b278d783ac622cdd7bd0699811e7b5e5195e5f9ac1dc86c9ba70086e29d57a0a4d4c02aab541458d1112f81b38a532a8076ffcd8a16e1efbe974e4")
	p1.Deserialize(b1)
	pubkeyList = append(pubkeyList, *p1)

	p14 := new(groupsig.Pubkey)
	b14, _ := hex.DecodeString("3c5d1dba47c495ce10a658e9e5c67151835eced353c0024dcdfe1f925fc92a760258ddb993dda493a8d33a60152337067dccdafb4e0f2f9eaea46490da091923589813549dcba58160514b0c0b0c1a1d920d431c2152764891d976639e1983f45849fa357006e38b8f41c19218a09adfe23940528b674e05f7e11e7a8078bd66")
	p14.Deserialize(b14)
	pubkeyList = append(pubkeyList, *p14)

	p4 := new(groupsig.Pubkey)
	b4, _ := hex.DecodeString("0e7197971858b9ec6789ac21a779dce3bee34151c918d931f18b2be84773ee0c141d1d20415d10bb6b55910ee0271311b4b1f33274c47fab0966c3606321f7153b8d58d883fac1483e2ab739a878a0ede52126e1b14dde507c07ecdfcc631d0f67646b715a09036081a7c310a9a3cdd0b8d3de4b5dbc2bba4f202cfb213f15a6")
	p4.Deserialize(b4)
	pubkeyList = append(pubkeyList, *p4)

	p6 := new(groupsig.Pubkey)
	b6, _ := hex.DecodeString("862bf075d1ab229f16f4c13b56bbf193bd86ac9b1a1b1ffabea1a528a0dfd0e3339b0c74e816674f613fd87a3511322e4583940e36780e466d7f7e0a3ba63f132f7ab281b6c2dde5914e2524ae3258329c6bc01aa94e48883a668c9dc509aeea895d434f3b5dc255d361e3fc61165e492315c626c0c2a96636052a8e58c44f47")
	p6.Deserialize(b6)
	pubkeyList = append(pubkeyList, *p6)

	p7 := new(groupsig.Pubkey)
	b7, _ := hex.DecodeString("6004c2bf80ab851002d6b0fa36813d0af1c3f7566fd828736a92536ad3f7cb1a8f40a4b620ce0e20ba12e4bf589b5e2ca36080938b085a8857b8f0e3907c37b2708fa3a2ccc0501510dff11df85747d14e139f5ce64896f082e855dae71db6d338f79e656feac8afd30166230712aa61ccb6245dc2ee8ad018d27e6e8cb06128")
	p7.Deserialize(b7)
	pubkeyList = append(pubkeyList, *p7)

	p9 := new(groupsig.Pubkey)
	b9, _ := hex.DecodeString("5a1a71596a3986792d52a35a7dfc592679f5b86b89c72cca6d8339373539d7e777d885eac730d03c32b6ee921aa1d72f30ac855248a57f2cb4db22e9771e0d21436a8cdfe2c4ff75e023ab9062b3ac1679e0e93afe24a3fce6d236cf309248954e43cb47d425aa9769f8f70849a9a6642c5e1ef2376b5631ad9917c341c5fbca")
	p9.Deserialize(b9)
	pubkeyList = append(pubkeyList, *p9)

	p13 := new(groupsig.Pubkey)
	b13, _ := hex.DecodeString("1a20140d07c7a5e634140e489750603865363c2f0386707c3437e172ce587b5b33b2978329f54f49ea37f2717cfaf70af5b42c16cb8f0af93eef1567564d78b22f87bc3b0f20799d1e4a067240605b317019e7a152c7370a579ae99cbdd2f04b5124124bef0a13b0fd0ca37f908367ddb6500b6f02601ce06644d394a15cb917")
	p13.Deserialize(b13)
	pubkeyList = append(pubkeyList, *p13)

	p8 := new(groupsig.Pubkey)
	b8, _ := hex.DecodeString("89d62d291bec395de1c2a8459d85fa56797d5ea26a583d1129d985e7349f1b2a3e6d58167d1e6cd1d0c08d9eccc4b6e3100851a096293d795a137b506d782d4c3d709d2a954b8cc0b49392be3bd279ae788860a087c39fca5213c2b1426b9ffc6d0e9b06bf081a7a3b5456d15677ce8800b5f571a4704da6c6c38bd5973f5b6b")
	p8.Deserialize(b8)
	pubkeyList = append(pubkeyList, *p8)

	p10 := new(groupsig.Pubkey)
	b10, _ := hex.DecodeString("8829b0479aa60da699a2a3ac8d8ad382ee3fd12cbe1e83dcdfbf07a7a644bc1e2a08f3d9c0f997c719d998810dd5a5662bf599e69641f0a0efd47ab5c387706064cfe9ef814693d8f09f2f94a6f5dd5c11072e0715a4d2bf9d108429353626f974af71e947a128efad2ce2eedfff0f0498df01ac24368e7dfaeca5649ae58771")
	p10.Deserialize(b10)
	pubkeyList = append(pubkeyList, *p10)

	return pubkeyList
}
