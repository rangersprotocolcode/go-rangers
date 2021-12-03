// Copyright 2020 The RocketProtocol Authors
// This file is part of the RocketProtocol library.
//
// The RocketProtocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The RocketProtocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the RocketProtocol library. If not, see <http://www.gnu.org/licenses/>.

package group_create

import (
	"com.tuntun.rocket/node/src/common"
	"com.tuntun.rocket/node/src/consensus/groupsig"
	"com.tuntun.rocket/node/src/consensus/model"
	"com.tuntun.rocket/node/src/consensus/vrf"
	"com.tuntun.rocket/node/src/middleware/types"
	"encoding/json"
	"io/ioutil"
	"strings"
)

const (
	robinGenesisGroup         = `{"GroupInfo":{"GroupID":"0xd588982453c1f6278564b44667573ec3eb6ce326a63e33e89b091cfe3de2a887","GroupPK":"0x608fdd9a055d2e688401220c9e3d0e9c2f26c354b3be07f1a06c4c6d769ae3cd38828f2c7b5b531fff8306a61fbdfa94e5eb59cecff7b28f4abba21f6037a2792d36d3544b2d7057405482a2ebde48f06e302dd1c91fbb6a47d393bb3f19f5660e2909d46e8445f857f1e63dff2bd10913663ca3c8a6bcbc64b0d49d92693251","GroupInitInfo":{"GroupHeader":{"Hash":"0x275d1ec7e591231e9de02a0d29cf465a5dfed5aea15feff938f85714bcc52296","Parent":null,"PreGroup":null,"Authority":777,"Name":"GX genesis group","BeginTime":"2020-12-23T10:31:42.523464+08:00","MemberRoot":"0x519750e82dd561f03eabae5ee7d1f11df0f6c14eb356757ad7bb7de7a51f1448","CreateHeight":0,"ReadyHeight":1,"WorkHeight":0,"DismissHeight":18446744073709551615,"Extends":""},"ParentGroupSign":{},"GroupMembers":["0x5437f9dd7171db9d04a8347dca5bf2b7789081631d79d2d7882c1774d2f4d123","0x2a17671c5a32175335fa098951ba50a9b4730aea7ecee86df6536297900f5b77","0xb1979dd362353f0b59dff76cb223d5660a024db628257693f5470dec18c93160"]},"MemberIndexMap":{"0x2a17671c5a32175335fa098951ba50a9b4730aea7ecee86df6536297900f5b77":1,"0x5437f9dd7171db9d04a8347dca5bf2b7789081631d79d2d7882c1774d2f4d123":0,"0xb1979dd362353f0b59dff76cb223d5660a024db628257693f5470dec18c93160":2},"ParentGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000","PrevGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000"},"VrfPubkey":["JevvUFyYiszl6wg3fPRA1zUBo8TKiphhjJy7Hy0nfcU=","8QZCqd7xDETZ1eA3QaQJUWsAcwYCwrM43UhyLGh9VP4=","mLYbDvkkPQFwnVRZCI59t1iLum4qdMIkEzADhpohvRY="],"Pubkeys":["0x81b7daee9550ac17c0446cb42bcf8cc87155fa36d641320fd351f5cfc5ce435a20457d5f3fe83d79aff4fb2cbcc0812242040d066463998eb5729866f1ce3979671fb242204c961cf2723321729aacbfff2c014f943ab8794785be47319efd76078d7cc657750903e24be9f09685fee361d321963fbc93075b9bdb18dd6f53f1","0x27bfa326a01e6640f61666801a6fc1acc37a18d5d4804c92e304dacef25d606f385fe9f050aeb4d5c9ee4c9d442dd535d65380aa26bfc33d92ab6007b91afdf41ddd68c1c6ab3e11154578eea8a364084fe8e4b263d675dd44ad8be223b75bcc0db0d4cbc4651eb2e5351ab5e01f3d3298227bbe342692582ab5afff39d237b9","0x2ece69e08094f7cdd3f19db009b907225438eea203ac19a27b2c7ac43350569105549567de059c3d247dd2fb9190f77f732d89f84d7540c752b7f837fabad6498c1570d42804397c1beee1876548b0c7abf804076dcfab61511dbd6a84b480230e90cfe8c2fa03e6e997fac2b5507f79eb6a2fdd215cc92bf343b7473b4b4df1"]}`
	mainNetGenesisGroup       = `{"GroupInfo":{"GroupID":"0x38f97eeeef60ac0e00b10f44541899455149781bf57b6823c6ade348e7c40f04","GroupPK":"0x80dfe956461155146363624d8dd27bea37d1139aedb5c28fc2fd22dc1aabfc8615a7bf893960860ae1dbdd3efb34d5d2e69a0311aac8cbacf010be87a89af694892b20701373ace414dbf4cfe0dccc7bd23e50abd4a9d17cfbde4876f59409a36ea645d05087a8a3c6557e56371dabeea03527cba8dc67d47c5e795892b0787b","GroupInitInfo":{"GroupHeader":{"Hash":"0x7b443efc6db49b0984f032f28766c6fca60c31d3da193949116c38792dcb4b1c","Parent":null,"PreGroup":null,"Authority":777,"Name":"Rangers Protocol Genesis Group","BeginTime":"2021-12-03T13:26:26.686415+08:00","MemberRoot":"0x05ca87f791ce5998a1399e2af3d605a4ece84457370ed2d3a1ae681fb26dde85","CreateHeight":0,"ReadyHeight":1,"WorkHeight":0,"DismissHeight":18446744073709551615,"Extends":""},"ParentGroupSign":{},"GroupMembers":["0xe733dda089d1127d2b442a323f616812f8550966a8af4a54d7849ad675463ca5","0x880f07e9cb8424d610aeccd4b2cc06a0a0b1ebf7a27fcc67420c84dc2cc64492","0x01d66f9b6fb802ce79516d58e98e31f170105b8c0523f96228a7b0f217605fbd","0xcd9872ddf50e5f058a6ad958cfe2149697e79fa7764b6aced358c0403e745c9f","0xd4c15e789357f8c860b1c55d251dd9cdbeb580ae1a132d83dc19a936f5bba41f","0x871915f7d1cd07767f4d7b9bbf8a5dbf91d89ff26649805ac767c13d75fef6fc","0x160b321d96c05393da1dedbea5d0905b9c7a41bc90da2779d77121ece33cb07d","0xd0815a3858f224b8baeee9c2e8d5732dbab12a7e97601e58d250dc23ce64fa7b","0x7ed35793cb96803d8a4c82be6a70f792de2420253fb38790e79bc5f4fb4d79dc","0x528e680cb4310cb0b745ef85f67d2a49bb03eb8622b0aa6cef65e3c3d7aff696"]},"MemberIndexMap":{"0x01d66f9b6fb802ce79516d58e98e31f170105b8c0523f96228a7b0f217605fbd":2,"0x160b321d96c05393da1dedbea5d0905b9c7a41bc90da2779d77121ece33cb07d":6,"0x528e680cb4310cb0b745ef85f67d2a49bb03eb8622b0aa6cef65e3c3d7aff696":9,"0x7ed35793cb96803d8a4c82be6a70f792de2420253fb38790e79bc5f4fb4d79dc":8,"0x871915f7d1cd07767f4d7b9bbf8a5dbf91d89ff26649805ac767c13d75fef6fc":5,"0x880f07e9cb8424d610aeccd4b2cc06a0a0b1ebf7a27fcc67420c84dc2cc64492":1,"0xcd9872ddf50e5f058a6ad958cfe2149697e79fa7764b6aced358c0403e745c9f":3,"0xd0815a3858f224b8baeee9c2e8d5732dbab12a7e97601e58d250dc23ce64fa7b":7,"0xd4c15e789357f8c860b1c55d251dd9cdbeb580ae1a132d83dc19a936f5bba41f":4,"0xe733dda089d1127d2b442a323f616812f8550966a8af4a54d7849ad675463ca5":0},"ParentGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000","PrevGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000"},"VrfPubkey":["THJSTfGTY4QAvc2GFeKdttU1Yt7Gh2J4sllbBPscoRk=","/Jlj0SE3AE6W/nZAmK4GHbm8p4sa3i2E2K52bP/UD0g=","bXvzKhjJxaLVFDlPxfSmB5dPLZbEsRbndaAsP/zm84M=","dAofvKM4dIdfaUaknZ9py8Pm3Sejt4CQR4PI7DMcIDM=","mI1Hdwv9x5rNYEXcUB+PZwXHG+5VRSYPF7B/PEJPJ0c=","M83VNJ0ygVDukeyQZT09wvszuQ/rOX/cIKXSZYHiMGs=","bx0XWMT9QaehgQfbANvddqwdrFysvbLTqf/KWAQLEW8=","FXdhSJVzFI+CSncvE5iIuVoA8IkubEC0pVl9j5GwPjw=","uP/fV4Gn0U/2vlAkuLti6OkOYOEd7TbTIdm9bXrCzpE=","NtoD8Wvi6CIhkwl1Mn0pb5CtF6rDoLHPYEjew9lek48="],"Pubkeys":["0x8b55628e8ccc5860d1f9bc4808946de061b67ae37c5b887a4fced24a7e9e555f4306224299ec72f23940e24f89722a49c2dd6c77320a701e507a95d93db0760633436782fb6b43b34c2467fe222f935cfb1df52ee17693c479461bafce78ae392ded0347317e28bb506fc89f0ce2721751dc1db2f57c76167da2836f5d0b1ff1","0x28f8c93c6ba1f9926ec18ba0632bb235687986047c72b9d1557acce75a9b5df888b672316b512b065835f5c576f7d3d4dbfd5176e325f18af7303012a4d9382e7741bebf6648a704f7c6cec8fa7f64fd1cefa536bc384d9c17946f9d56d3df41837f3163e5a49889092f8c6073ad7744c3f4b96b327ab36a07ae19453f9520cf","0x0aa49c19b3bb740aa7f09df1c45e66ed3edb90f13842592a547e3b75b96b705034173a6bc9942ee8db9e15a11e53d6c320bfe002a76b88ac587d166e8d9a416e2e6bb891d6a6cbf4fcdad2c0d0ebeefe4b0075c93f0951454c96a738af1f65445cd4cf628ff767982d7dda314c9f691004148a5a8c4eee9d48a7ff9f529c4b4b","0x43f5bb4fcce4d7425f4b28f272f79065319b727143ec51254979cc54190b616413960bc13d95c7c0f19f22de9b969b42bec3bd2b7091f1d1a252a78994363afd2a8b8788d1acfccaf1976cb2073902879f98ca671ec6c391fe927fe92ee687c9510d7ad9b6e0fffe8ff7a4a7c5dcce5434cfb0b2fc338d6e004cf0d0948e99fd","0x15e1aa6791bf01719f211812b59ba6e8f853377b8c224e4d01256c5854fac991138c454de1a72034cc1e54dce0f3430cdb320f1581f5da3f2a018ef4a4673374497656db78dd459f20b2be3d565e503be1120ea4527cd8edb202165d888119d2036bb028d0cdf891ce210b971eef40af2a27db4fd5b113d8e081eb2f31ccf1f6","0x42d3fea5ad6ffc4007e2994333dac0e0d755bc1784f0bcb5d76a88fee5940a5836dc31a84ecb51716195e4d6438df68e627df4b121afe96340aafd49a6f8ed237693dc55c8cb37ceebc66aa39d0ef83c1180030bad4ec242364401bfaceed58c572378c675d28ecb92c7f1c228b029c46ab8785a4b8adab97d263a62ca4e677e","0x74860aa0e71c1dfdef0cd07f29530e51c398a7b044b83c47ff260f741674b4295bd91d6aa4855064a1287e7f6970dbc2ec19e5a157306a69897d742a8db60f1f80566c20783168e0724bb6061400ad950d3038b5bff6197e4dd73f02dbe2537e319a39c97931446861762e21ccc415fd3bea749bfbc65d81ad9f133102aff558","0x15c96eb18f35138c202ce9ea9884ed055c3375a666c3dcd3f265b4c919f0e8927a212193533e90aa4408ce319bfe8b0279e3033efa2c48f302dd9e716c844ebb0b2a0ed66f07a863657485d8f0539c067a98d2dca64694546744bc44af4f3a06045bc2e68bf64f47a914a131a281a3beb634cfd2379a19254d1cb030f88b102e","0x4eb0aada6e76cff82cc7c0eb52e03135df79729779abd70082812821adcf2e31492a223bf11f81a08975c5978f50a65bc416ef28e9cf3c9f33a36ec494a24a84438c39899975171d11d9dac0f3ab45e24739dbbc9e9ecb27ce4ba940df6d75130272b5d1da0382da699ef84afcfd425e8efadf66633bc60571c260454e2683de","0x73882be0f806f14fb13def125ad2f841c6249bb49eb340661e72b4a561951e811c25619e0132c317a831561063cd32029d6c23c1f4540ce27ff856d5d054472c6e0f75453910b18bcfb88efee06512ba669081bec16da930a2c0954fff7a0cd037df78aa91d1579d3821ae9bdaa191f81ee5b397068c50e722ea379050a05ea4"]}`
	robinGenesisJoinedGroup   = `{"GroupHash":"0x275d1ec7e591231e9de02a0d29cf465a5dfed5aea15feff938f85714bcc52296","GroupID":"0xd588982453c1f6278564b44667573ec3eb6ce326a63e33e89b091cfe3de2a887","GroupPK":"0x608fdd9a055d2e688401220c9e3d0e9c2f26c354b3be07f1a06c4c6d769ae3cd38828f2c7b5b531fff8306a61fbdfa94e5eb59cecff7b28f4abba21f6037a2792d36d3544b2d7057405482a2ebde48f06e302dd1c91fbb6a47d393bb3f19f5660e2909d46e8445f857f1e63dff2bd10913663ca3c8a6bcbc64b0d49d92693251","SignSecKey":{},"MemberSignPubkeyMap":{"0x2a17671c5a32175335fa098951ba50a9b4730aea7ecee86df6536297900f5b77":"0x81765735aa48fa38eab581a961630adb459bca8b33a540faed3abaf5baa0a4e44ca776e946ffe9b857421c21062ac3837684b72ef36d174d140f0990972f1da94509dba0edb65dd72b647143776b0d6c25c3a47d0419581d2730bae0e3c0ca34450cfb9346b7fadd6737ffb2d78048ec0360f2a57eb8db870d79d8e51ca31817","0x5437f9dd7171db9d04a8347dca5bf2b7789081631d79d2d7882c1774d2f4d123":"0x6a2a1db50572d6f026ab8274357e961229dbc0dc73e6e5e746fc08237875dffd725bb28a4e4e7e4b5fcc299ab3ff2fb18e64b1822a44b4486b2045d34b5b7e2d8a6a90742f02122be4efd09c6fb454ce606ee80666fd75f792c0e96140bc6b9283b90f32b5916bfa721e18422f5255a73ca9330151367bbc1731ff797a5c7496","0xb1979dd362353f0b59dff76cb223d5660a024db628257693f5470dec18c93160":"0x8cad31d6d8a33b3de6cf964c23aa66e17e56b5aa5914c9d3702871a2e2b41cb63c34a8741ac08ac8d64147b3b72b4ea26f59e6257d26ff59dc0e5088ea0a7e3188d20929c11b012b1e6ed2c2ef4e4b29b9ca6b4e1bd5adff6b899fba6e9df731837b85e47e53969b414839ee28efc4a245e7489726b9c9e5ff2d50fa1e676a4d"}}`
	mainNetGenesisJoinedGroup = `{"GroupHash":"0x7b443efc6db49b0984f032f28766c6fca60c31d3da193949116c38792dcb4b1c","GroupID":"0x38f97eeeef60ac0e00b10f44541899455149781bf57b6823c6ade348e7c40f04","GroupPK":"0x80dfe956461155146363624d8dd27bea37d1139aedb5c28fc2fd22dc1aabfc8615a7bf893960860ae1dbdd3efb34d5d2e69a0311aac8cbacf010be87a89af694892b20701373ace414dbf4cfe0dccc7bd23e50abd4a9d17cfbde4876f59409a36ea645d05087a8a3c6557e56371dabeea03527cba8dc67d47c5e795892b0787b","SignSecKey":{},"MemberSignPubkeyMap":{"0x01d66f9b6fb802ce79516d58e98e31f170105b8c0523f96228a7b0f217605fbd":"0x0a1acf3c14dbd6423cf3f92c912a7ce56e0c3389dac8ead665dbf88d445be75636b41ff174939885adaa19dec41044aa53e46ff80594e2709b3dad4ab56208db4a9979d43c4d3b27a3187f58ce62c99a9e5c3bcfa2131a5110830e9500e963357443e25bd59c116f2a4de6df1c90ee1d808243eb578a048b49208bc925c2eb96","0x160b321d96c05393da1dedbea5d0905b9c7a41bc90da2779d77121ece33cb07d":"0x1ae731eacacf8d217a0f561ad3a156d2af322badd2ec5af55059163a090f75a782241b00b6768b6e83e81c0f29918e3dda731bb856a6cd930d8357a7cdc9ef7781af88445688e93b500d807507c18a12ae93cc4bc0bd724b34958d36f0bf0658026bc0a375b4689c350148ac6c05bdc315590c1720333e23e6150e4e6e021b73","0x528e680cb4310cb0b745ef85f67d2a49bb03eb8622b0aa6cef65e3c3d7aff696":"0x3c0f19a69576d7ab94284e5921fc21b5b990298a11d3fc0653db13d2b2fbe9107b87e7ef636e2e370cbf307abe1edb8f618c5121811683cb838a68aee3386ac2605571f51070e620ec4e8d8d9916f0989f28d81c8cde36c0a58b9f685615f2c952c858a1a16b7e55b1d29e3f17ea7800c40eeca638a4ae5e5c8a7c3cc3ca7ea4","0x7ed35793cb96803d8a4c82be6a70f792de2420253fb38790e79bc5f4fb4d79dc":"0x025c9c95d45d24d1a9a41e58726fbaec058ba923393d0d78a85759b1945a84e466999ec6626331ad83660e087c2c563d3f1f559de07df26029dc5f61d5f63efb4cd6f50e83559e0f2a91a8c0c6a4726ed3a62ef713289d6b1de4d3aee72ea68e2d3553292a9abbe31c69363b27847bec1198648edde48ece65d7e8364ace0c27","0x871915f7d1cd07767f4d7b9bbf8a5dbf91d89ff26649805ac767c13d75fef6fc":"0x71357e080ea2b7d8342dcb15605f381409425c3e5fb74b70ee557ae300e06fd877097f2d1b344804da5dbce07a3b4b7238190ee59bc797efe490f37f7e7eccca6b4ac22b6233e4c893de7a03297ee0b8e36bbbc38579cd4d9ae04b3de402cb780688b23ba1f75a1bc917c8545fa56109cea1bcef43722fbde62b7fa1125b86c8","0x880f07e9cb8424d610aeccd4b2cc06a0a0b1ebf7a27fcc67420c84dc2cc64492":"0x3b397a356b5347b38f279392c37b09ede56440a0e4f6c8707285ac6141b2464972a561f9a51955510a2a4c3a35b2ed0429ce92193ee65249d8ff0200f44b78ad54a9af1f0528e77bdc295f47b1e37034c74e1fe6586b541512a3beef14725954402dabed5b880b1d60acbac585e4f3c381c339332e4b39e9f451bb4574702cf6","0xcd9872ddf50e5f058a6ad958cfe2149697e79fa7764b6aced358c0403e745c9f":"0x3f305accedaedd673f62741d0ae07a04016e30f99536c81fcbdc22d292a08cbf89f896267001df194951da3c07f0fc5ad92146b4429b7769681e9920d43d31bf3d42b75ba704b63172504577b902d39e7452f099faad9799412c393be8cccca0746bc5798512f703ecf055beeb841be1ee88aa19fcbb16fae22f526aee6dcbc2","0xd0815a3858f224b8baeee9c2e8d5732dbab12a7e97601e58d250dc23ce64fa7b":"0x5a455951914477118265e9c4a219bf75f9be001930de3deb54651e853c610f8f1a9ebab293813cdda19b0322668d4e8d274add94107bfe5e03c652dc3dfcb43d07274b9b12750496350d6b1867320e2fc48ba5db711893cddf5feb9675eb1e896f67000eb1b9e5ec80b0772fa1692272bc79ee05a58905e6ae68da13bfd28ded","0xd4c15e789357f8c860b1c55d251dd9cdbeb580ae1a132d83dc19a936f5bba41f":"0x4e0796037387f837bde9aa89e55bb83cca4e43a113db7d866cef4a9f1492ce8761b91a860d9d6589cd8eced76d4e183cd1252e15fa4a503c78f671e11d37e6e4155349fa85b9de5777d2a884a51de518f9216d9e00df756ecb6acb78913245d1877bef3692086ad1314785df9a0fad3175d07e016c33dc91049a52277d2faca8","0xe733dda089d1127d2b442a323f616812f8550966a8af4a54d7849ad675463ca5":"0x18774dd63298a054f5e4a8035bf0fe95f9d11fe49334ffd1b66cb4adcda6b099124c7d20e5dbf354302a9cacd85f8c8b79709218d2725e2ec1ad4dfd590ecabd0cb994ab0e8388f742755eb13af63c1477830dd814630e5ac1c568d00939c91f833fee65d8887a978a818c36fc95494120e478b230f5fce15f52f64055d91881"}}`
)

type genesisGroup struct {
	GroupInfo model.GroupInfo
	VrfPubkey []vrf.VRFPublicKey
	Pubkeys   []groupsig.Pubkey
}

var genesisGroupInfo []*genesisGroup

//GenerateGenesis
func GetGenesisInfo() []*types.GenesisInfo {
	genesisGroups := getGenesisGroupInfo()
	var genesisInfos = make([]*types.GenesisInfo, 0)
	for _, genesis := range genesisGroups {
		sgi := &genesis.GroupInfo
		coreGroup := convertToGroup(sgi)
		vrfPKs := make([][]byte, sgi.GetMemberCount())
		pks := make([][]byte, sgi.GetMemberCount())

		for i, vpk := range genesis.VrfPubkey {
			vrfPKs[i] = vpk
		}
		for i, vpk := range genesis.Pubkeys {
			pks[i] = vpk.Serialize()
		}
		genesisGroupInfo := &types.GenesisInfo{Group: *coreGroup, VrfPKs: vrfPKs, Pks: pks}
		genesisInfos = append(genesisInfos, genesisGroupInfo)
	}
	return genesisInfos
}

//生成创世组成员信息
//BeginGenesisGroupMember
func (p *groupCreateProcessor) BeginGenesisGroupMember() {
	genesisGroups := getGenesisGroupInfo()
	joinedGroups := genGenesisJoinedGroup()
	for _, genesis := range genesisGroups {
		if !genesis.GroupInfo.MemExist(p.minerInfo.ID) {
			continue
		}

		joinedGroup := joinedGroups[genesis.GroupInfo.GroupID.GetHexString()]
		if joinedGroup == nil {
			continue
		}
		sec := new(groupsig.Seckey)
		sec.SetHexString(common.GlobalConf.GetString("gx", "signSecKey", ""))
		joinedGroup.SignSecKey = *sec

		p.joinedGroupStorage.JoinGroup(joinedGroup, p.minerInfo.ID)
	}

}

func genGenesisJoinedGroup() map[string]*model.JoinedGroupInfo {
	var joinedGroupInfo string
	if common.IsMainnet() {
		joinedGroupInfo = mainNetGenesisJoinedGroup
	} else {
		joinedGroupInfo = robinGenesisJoinedGroup
	}
	splited := strings.Split(joinedGroupInfo, "&&")

	var joinedGroups = make(map[string]*model.JoinedGroupInfo, 0)
	for _, split := range splited {
		joinedGroup := new(model.JoinedGroupInfo)
		err := json.Unmarshal([]byte(split), joinedGroup)
		if err != nil {
			panic(err)
		}
		joinedGroups[joinedGroup.GroupID.GetHexString()] = joinedGroup
	}
	return joinedGroups
}

func getGenesisGroupInfo() []*genesisGroup {
	if genesisGroupInfo == nil {
		genesisGroupInfo = genGenesisGroupInfo()
	}
	return genesisGroupInfo
}

//genGenesisStaticGroupInfo
func genGenesisGroupInfo() []*genesisGroup {
	var genesisGroupInfo string
	if common.IsMainnet() {
		genesisGroupInfo = mainNetGenesisGroup
	} else {
		genesisGroupInfo = robinGenesisGroup
	}
	splited := strings.Split(genesisGroupInfo, "&&")
	var groups = make([]*genesisGroup, 0)
	for _, split := range splited {
		genesis := new(genesisGroup)
		err := json.Unmarshal([]byte(split), genesis)
		if err != nil {
			panic(err)
		}
		group := genesis.GroupInfo
		group.BuildMemberIndex()
		groups = append(groups, genesis)
	}

	return groups
}

//ConvertStaticGroup2CoreGroup
func convertToGroup(groupInfo *model.GroupInfo) *types.Group {
	members := make([][]byte, groupInfo.GetMemberCount())
	for idx, miner := range groupInfo.GroupInitInfo.GroupMembers {
		members[idx] = miner.Serialize()
	}
	return &types.Group{
		Header:    groupInfo.GetGroupHeader(),
		Id:        groupInfo.GroupID.Serialize(),
		PubKey:    groupInfo.GroupPK.Serialize(),
		Signature: groupInfo.GroupInitInfo.ParentGroupSign.Serialize(),
		Members:   members,
	}
}

func readGenesisJoinedGroup(file string, sgi *model.GroupInfo) *model.JoinedGroupInfo {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		panic("read genesis joinedGroup file failed!err=" + err.Error())
	}
	var group = new(model.JoinedGroupInfo)
	err = json.Unmarshal(data, group)
	if err != nil {
		panic(err)
	}
	group.GroupPK = sgi.GroupPK
	group.GroupID = sgi.GroupID
	return group
}
