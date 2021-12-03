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
	robinGenesisGroup   = `{"GroupInfo":{"GroupID":"0xd588982453c1f6278564b44667573ec3eb6ce326a63e33e89b091cfe3de2a887","GroupPK":"0x608fdd9a055d2e688401220c9e3d0e9c2f26c354b3be07f1a06c4c6d769ae3cd38828f2c7b5b531fff8306a61fbdfa94e5eb59cecff7b28f4abba21f6037a2792d36d3544b2d7057405482a2ebde48f06e302dd1c91fbb6a47d393bb3f19f5660e2909d46e8445f857f1e63dff2bd10913663ca3c8a6bcbc64b0d49d92693251","GroupInitInfo":{"GroupHeader":{"Hash":"0x275d1ec7e591231e9de02a0d29cf465a5dfed5aea15feff938f85714bcc52296","Parent":null,"PreGroup":null,"Authority":777,"Name":"GX genesis group","BeginTime":"2020-12-23T10:31:42.523464+08:00","MemberRoot":"0x519750e82dd561f03eabae5ee7d1f11df0f6c14eb356757ad7bb7de7a51f1448","CreateHeight":0,"ReadyHeight":1,"WorkHeight":0,"DismissHeight":18446744073709551615,"Extends":""},"ParentGroupSign":{},"GroupMembers":["0x5437f9dd7171db9d04a8347dca5bf2b7789081631d79d2d7882c1774d2f4d123","0x2a17671c5a32175335fa098951ba50a9b4730aea7ecee86df6536297900f5b77","0xb1979dd362353f0b59dff76cb223d5660a024db628257693f5470dec18c93160"]},"MemberIndexMap":{"0x2a17671c5a32175335fa098951ba50a9b4730aea7ecee86df6536297900f5b77":1,"0x5437f9dd7171db9d04a8347dca5bf2b7789081631d79d2d7882c1774d2f4d123":0,"0xb1979dd362353f0b59dff76cb223d5660a024db628257693f5470dec18c93160":2},"ParentGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000","PrevGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000"},"VrfPubkey":["JevvUFyYiszl6wg3fPRA1zUBo8TKiphhjJy7Hy0nfcU=","8QZCqd7xDETZ1eA3QaQJUWsAcwYCwrM43UhyLGh9VP4=","mLYbDvkkPQFwnVRZCI59t1iLum4qdMIkEzADhpohvRY="],"Pubkeys":["0x81b7daee9550ac17c0446cb42bcf8cc87155fa36d641320fd351f5cfc5ce435a20457d5f3fe83d79aff4fb2cbcc0812242040d066463998eb5729866f1ce3979671fb242204c961cf2723321729aacbfff2c014f943ab8794785be47319efd76078d7cc657750903e24be9f09685fee361d321963fbc93075b9bdb18dd6f53f1","0x27bfa326a01e6640f61666801a6fc1acc37a18d5d4804c92e304dacef25d606f385fe9f050aeb4d5c9ee4c9d442dd535d65380aa26bfc33d92ab6007b91afdf41ddd68c1c6ab3e11154578eea8a364084fe8e4b263d675dd44ad8be223b75bcc0db0d4cbc4651eb2e5351ab5e01f3d3298227bbe342692582ab5afff39d237b9","0x2ece69e08094f7cdd3f19db009b907225438eea203ac19a27b2c7ac43350569105549567de059c3d247dd2fb9190f77f732d89f84d7540c752b7f837fabad6498c1570d42804397c1beee1876548b0c7abf804076dcfab61511dbd6a84b480230e90cfe8c2fa03e6e997fac2b5507f79eb6a2fdd215cc92bf343b7473b4b4df1"]}`
	mainNetGenesisGroup = `{"GroupInfo":{"GroupID":"0xf8c4e4b03c091a76109065ce13035ff7e2bc7bfadb6ede8a9b9aa3927806c3e0","GroupPK":"0x2a7636f27fa9f38e70e4ce71134c00af9850d9788016e8aa694e2b107dfa75444b0c893af86a14b4d426c349ce933b863cc02e91f976999f7f8437b617af99aa1fa24ce78d93b6418f22048976722b6bf2a7310fcbf96ee6b8d138a0f00e79995187d043b6aa290548ccf62488bbaba9ddb64d78f913940ef71a97505fd8f6c2","GroupInitInfo":{"GroupHeader":{"Hash":"0xaae596fd903c7f36ac02f6c97e5930a1ddb691795eefa98ae16f44089f0ad833","Parent":null,"PreGroup":null,"Authority":777,"Name":"Rangers Protocol Genesis Group","BeginTime":"2021-12-03T10:08:54.72498+08:00","MemberRoot":"0x1b1a74baeeacd8b78a96ac1e94e5be83fb517c848eab2047a6625de7b53262c4","CreateHeight":0,"ReadyHeight":1,"WorkHeight":0,"DismissHeight":18446744073709551615,"Extends":""},"ParentGroupSign":{},"GroupMembers":["0x7b4a904a6c7aa9b75d1cfe18d93c421ec214b17037283a8e7690b5445836421d","0x00e663f4e4f6ef09b0febaacf55e13b2408db978df935b908ef7027358e654f8","0xbbd46b3e775c245343b2bfceeb9439bb824220726cbd9ed2229360496b663304","0x9b7ad7f53b230ba2919722497da8438898e3f574726e20230bd690e9e9d9d9fe","0x90d301d65727c2ab76abe5fd5020036e5b9065fb1d19022744c30a1ef5d4945a","0x1c55fb0fb9fa080f6885a14088f9efd54067f9de43fe7d05fb186c54cdd8090b","0x22c4dd15964495f7d0c51b7c0d1af93f993c362fdd6d874df47fb4c220166267","0x14398c018719479dd1ad124d1b991e4420c01d33c1e07c91f662b775f3e89f82","0xe74d7ae7ab16efd47a4828ccf9e38732a724fb06f99f65bf41eb243167cd37d8","0x4306a858b6e579fc2a54584ea13ebb474ead98e5269d2d8dfc9caf4d73acc772"]},"MemberIndexMap":{"0x00e663f4e4f6ef09b0febaacf55e13b2408db978df935b908ef7027358e654f8":1,"0x14398c018719479dd1ad124d1b991e4420c01d33c1e07c91f662b775f3e89f82":7,"0x1c55fb0fb9fa080f6885a14088f9efd54067f9de43fe7d05fb186c54cdd8090b":5,"0x22c4dd15964495f7d0c51b7c0d1af93f993c362fdd6d874df47fb4c220166267":6,"0x4306a858b6e579fc2a54584ea13ebb474ead98e5269d2d8dfc9caf4d73acc772":9,"0x7b4a904a6c7aa9b75d1cfe18d93c421ec214b17037283a8e7690b5445836421d":0,"0x90d301d65727c2ab76abe5fd5020036e5b9065fb1d19022744c30a1ef5d4945a":4,"0x9b7ad7f53b230ba2919722497da8438898e3f574726e20230bd690e9e9d9d9fe":3,"0xbbd46b3e775c245343b2bfceeb9439bb824220726cbd9ed2229360496b663304":2,"0xe74d7ae7ab16efd47a4828ccf9e38732a724fb06f99f65bf41eb243167cd37d8":8},"ParentGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000","PrevGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000"},"VrfPubkey":["YM22FSfLFprm/WpVo0xiC2eIHrlES4MX9fjFe4DiFzg=","/Fzw3VVFQ1fteoUcDQISdP0VnwdK44d3AM/xZL0+wBQ=","YQIwvXq/YAcBTVjSzbEVNUi7ts/ub8FubCETVyserCc=","25SLM8iPeUu4Ociqon2bF2K4uRwZzsjmhk5cTi74kXw=","ZD9aX8OcWkDD2xFyiZl1w5e8gAbEssgHSaQrZmU/jdE=","7Bd2wAUMRyAEudKyYMg64dn8/hU5KGaoDnh/ZXFBnCE=","Xn03tOZmMH6Xlaa8RtjGPYRsoIPX1L3nIytGTOMP4ko=","abtFQi5xSerKODd3bPt/7pBZ7++dmCULhISO8sr1gqs=","9e0fRdBxumcKXTRxuIL+8t9wqtR5XM96Umv6gHt/xoE=","YFyftZaJ/7aesV3IW2LNHPKHqh9C9EikWasRQej702A="],"Pubkeys":["0x85002f2fc2e8a615c09a1aeaf82c8af13f7a6d88817325eaa72b264dbabdf801184a91c23e5e806dc2aa4bcdae743730eb1a5f4cfbfedb85c54d356ec245dee73059444e89a2159f84935d96edf20ebbd26770ebcc31ab9d4ed189646d9a487825a21780f3ed61505d0a800d4d6c8d44eff8a0f14a18693edf1305ed4bb43534","0x558320c39c69ee36c88cb9afa4d122ce3f86fa25611c4b620d261e580efcab8226e9f8eb1e204cbe5eef298ad1874852f7cd709a288b62e140b145fa4b2e2e88296f2cfd87749a2c3d74e4f56ee67aa5ccdccddb66b5381a866772b46767f9ca80012c8615f2d8b4a20cb6611d8985961a773e66e6970b108fa89d7c7bcd5f7a","0x0585fd24090d3604cbe106abb6e85063917595c2729229b697d0ae4e123ea8e37a2cce42d6b58ad5af184840ad6d95bbcd1e073cc348ad36eb427a6f3e04574c8418385b85a39f6dacd97a45c22f723a088908a60b111a55c15c769810acc8dd1df22e4a9d0a06cc502a082e245a0966dadf2620ec7bf85061293c0b0f3e2897","0x697fe35a633421b3fc8993466c912af1d02de7b259771633cb1343d6a3bef05d3a99c7fa3517da4631ee7bd2ac279a22476388ae7e697ea92ddf5aa3d0e10c3726b735b160f626280f558117ea28066af5adf6a0362afdf15f134d6360a670fe819fa832e435a64f714ea9880e14439a28cfed1a30d3c77de458e89a4989aa57","0x323c58a3e3f82d13fdbfb0ea4f60cb7e5460b9b15a32ca5640fef5381bb0ba2669d9a17f1346cc58c382f12ab8ea01d1802f50b7ddb7e84e814409979d2cd9b836429e676ece2346ad2dbf6e4981269b4d862bb1e098ee394505e461ff843ddc0cd2c588217c02c309c2bd5ab41d694d26494cb26150d56c8231fa6acc5b4f0b","0x2881a9d38af1cfc244343a957c7ae0b54b2fbf23257f73a836f063c5fa61d42560f1312f7eaeca83a5d3cd9dfcc6d94a7ed7d0f93e63581a2768daf6b30e268f2e7cc1adf616187e0d4ca3e6c84f6eed0e8900047ed8a5002b75833a555a614408a3951f9e0e9cb02af7eb8a1538f2cb51038954b7aae58eb314422b5916bdcd","0x7dd0ffac4e8eb4d130d28a167215df9f5f55e3dda1cc93e06364ad7cbd5d4b2a662e7efba82c3d79b3a5a285b060c78256d3ca916d168af429d614c42d2906c513cf932c4d7162d0d5ae2cd4fbaa9db8975034fcc5ccf414daf3d8701cba513258597fcfdd15beb7b28ad78151f596c867c16818fe8d625be2a29020014a2a2f","0x1a15b94eda22c784a3e0e593cd69c726ec38256dff7c3103252cc64b57beac71530ae97c78964edbb568306094a93bf6556969e083673e7087f6ee7bcac0abc55b4bc1ebe425a182fbfe2476d4cd6ee5d799394cfb6dc88f5ac25f09378dae1b333afcc0168d97f592ddde398b6de0e2152f971900cb989f139eb4dc9132ff55","0x0faa08f54ea9440210d193028afbae3434b42e55e32ece2daf786ac3f142e71b0c8494a2e1df665cac52024c7420884e4f5a23413a07b8a612fd8aecd09ab8113fa0fc5e17298c6108d01c3855c47fb381a497e9024d44aa157381912a1592b77f33f7f7a978b2997645e0f1919d358ec1d553acd92997275a524b11717e00d5","0x7c6f09ad2d7e75aaea44c4173c8e2536d137f35c06ca3f67ce328ae692a6c1795951116cc73e96cd5596d834ec1019e0c5526748c23b8e75fbec1d96167a58715287270daf5fcd699e7836f282de88d039666f8d481dcdf01fd10bd60aac926d88aaee1c8efef82f03a9078c2f1e003e7eff226ea39c8c60349cc66e7f7580dc"]}&&{"GroupInfo":{"GroupID":"0x1c0164fe4b56206321780ced0306f2b033b7d0140257b7d594aee0e1d7c4bb14","GroupPK":"0x6d22919147272e5b354f081dcca849a9785011cc3d67ac7a54c0002481ffcfdf56b777bb2a327e17d08f92b350253d99dfdbcf2ea4cc49850f6a4ac5a8205df43723f82057dbb10bd98d0b252e86b69d2eea60a167519f92f0326f73438d02c75d187defc44076fb48fff2c591ada4d4e94358ce9f65c0f05b56c94f73e89447","GroupInitInfo":{"GroupHeader":{"Hash":"0xeec5ec8b6e74d9d03ff6e0cfbae37bebfee90580491616df795b22d887603ce1","Parent":null,"PreGroup":null,"Authority":777,"Name":"Rangers Protocol Genesis Group","BeginTime":"2021-12-03T10:09:41.80103+08:00","MemberRoot":"0x635f56cb806ab46e83ab5dc05b332a02981e308344b8010e7003982729674564","CreateHeight":0,"ReadyHeight":1,"WorkHeight":0,"DismissHeight":18446744073709551615,"Extends":""},"ParentGroupSign":{},"GroupMembers":["0x245f61a18215af0d2531b8e6921b13b52cd58fa43c2e19cd75c9031040ecc7fa","0x77f144dcc7a25692a70ac7324504e764a3f6db0b3f954de87e8dc1a25467fe1a","0xc8ad0be14a695432843e76485644eb41618c0307ceadd49fae57961f8ef716d1","0xab2d103f32136d9d635eeadfd82d6ac340788d3bf70862270ee04b82e94c79ee","0x138f0771a33357766ae663f4889e8b8be2654ac401214557c407868504051455","0x6ddc3604fbb86696627828f7e9f33c85df6d66e19ecd12f29d7b095da32f3ed4","0x4d2300e3c25fd4214072e4adcf599abed366b5be1cea8de3829a80d48eca4205","0x3748f337f5915aaa3bc150fa22c7aebd1e3e397e3f61fb501eeacac1a268165e","0xae003a027036ed72eac4d367fffd6979993b62f94e98a10c8d189ee462d19d84","0xbff465dd49992a79958dfa3c7e72a994558885c9d63f6ec5325701a74dde732f"]},"MemberIndexMap":{"0x138f0771a33357766ae663f4889e8b8be2654ac401214557c407868504051455":4,"0x245f61a18215af0d2531b8e6921b13b52cd58fa43c2e19cd75c9031040ecc7fa":0,"0x3748f337f5915aaa3bc150fa22c7aebd1e3e397e3f61fb501eeacac1a268165e":7,"0x4d2300e3c25fd4214072e4adcf599abed366b5be1cea8de3829a80d48eca4205":6,"0x6ddc3604fbb86696627828f7e9f33c85df6d66e19ecd12f29d7b095da32f3ed4":5,"0x77f144dcc7a25692a70ac7324504e764a3f6db0b3f954de87e8dc1a25467fe1a":1,"0xab2d103f32136d9d635eeadfd82d6ac340788d3bf70862270ee04b82e94c79ee":3,"0xae003a027036ed72eac4d367fffd6979993b62f94e98a10c8d189ee462d19d84":8,"0xbff465dd49992a79958dfa3c7e72a994558885c9d63f6ec5325701a74dde732f":9,"0xc8ad0be14a695432843e76485644eb41618c0307ceadd49fae57961f8ef716d1":2},"ParentGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000","PrevGroupID":"0x0000000000000000000000000000000000000000000000000000000000000000"},"VrfPubkey":["2DIloyPNiAfQjWxHXIA7huxiBSJMRUBmLI/p4fBGN10=","skKfhAUjHsTeKF+AmhjsxftDGdtKJibRc+eaYawOZfQ=","VgLuXGCzHV883wrkgMRnbASCXmbgGaCRefaA3nO5GcA=","PrrLRz0V582tTgPHphHm7j79JSdJbe06IsGLqdAquBA=","7y9PYF9Pi9/Ibd8mCkX19SRRIaSXtXKFS8VA8suU9mI=","7daj/GETnCpcW/ArnMDHibrujPC+wdaJbbR1y5+4WsQ=","YuSblTxl7Xtooha1JAXMu/urrmxTQx6li6pkSsrlZoc=","FROaYACRmPW7gpvhzpMDfXRYcxwXebEAZswkhGBSvJc=","VxzKknLaFV6DIR/Sl1rsg2IDCZpUd+XA/hJFNHk7tLg=","NEuDtJhgG6BbR5/R8hzpLRoQrohaSSAOhrXeWKxK8ys="],"Pubkeys":["0x7bc7d1769eaa42ad7d426ad3ac91ceb4d151bf149803e9fac558b462ff88ac1275802768e37331c0f2b8dcb1803383231033a007ae51b70d01eea01b8200d3d71c864906733bde01162db2a8620ed6624636ce629bb69502a433fce52541120d46155faa43b51a17d0d40160fee700b8ec52ef7c9ad7f23de9a59d2925763c71","0x6beeb23e21ea97fb62461b22ea65d4cfb577a0129db35d45f5ae5941ea44123f79c58e3ec6c539c8aedf479af8173c5589ef09bb0add4aa883c4d38d5c02469d264703a622acf754b22003058e9bd83cefd75cca2811906f6085bb7cfc8e051b7e641d36882f7b82ff8b9c3d6f3950b5cf89c25fb13bfd8174543b465c550f3a","0x4dd9ad9c8e900f052512878e334403db9ae8d0de01afc8fc4b6e6ec013126dd027bd155f6f42be2ef9a4a0a4159c49bd7899633585087f5d474c1aba5e104b6e34e6512a1b9b39bf69eb954d3856dac9eb333dcbc72eb2783aa004f9450080d37d44a47cc00da8083614bbdc29b29740f612a3aafa3fa04c00529a7926d08138","0x1eca1a994585d9c7e0efb2ba46ce280ef83c204c586fc1a630be1e3558fba9a80b2f579ab2ee097db4e3f333b72524e570758d0b2e437a0b0a39c82e8fbeabc332feecfe0d51c6e16b090fb22992ec3bce51a2c71cc3804ace3d7b32442c212d385e66aaad3849071c506e1b69d1b6d40da79028b5bac1137481dc2838f31096","0x57c58bd95b56557e8997be6892b1b32ea4cde082db496ea84c2559964396b096308f40001bf1e62ce3aab249f1ae9b4c0822754f83afe6a5316704a60db1c0981d8077211e29133079fa61020e861d98c6e5ee7a5c4a25895034270198a718745dd6f138893790cd69e1c73671dc4b186c5ea85bec5bf9c3c63babaccada2557","0x130f2e35397559d321e10a93e389076ef05797e86cbcf087f815db8c2b3da3fb5bfbf9f06f56f50ef49ae3a3d1734acd81007512e914cc62db9a1b6f5d28ff1824343b5796989aae0ddd571e79070be9374da57c14f0fd026c939aaa3e3ec1a84a5c0dab3ab11aebcdf1e4417d1d486bb358af823ff85475da78d2e51bc63a1d","0x33307ed96774992307fcdfcd87dd5bde108f5f994f74569a8bbb4070548c05b85200cf9d2e1bdc61020034f3c908ef330e2ad7067a28d0b96a303fa12d451303231b3b135092e637e11736c84b5847aafff59241f0abdc4b4cc82c11ed53ecf38bd5c8133781aed5fa58234b5ea4a297defb30df0279d8038fafaf8c33137271","0x4e368c924728c8c2caa647a1321f3b0206cd9f8ba0ffbf8e1215dc76fd9098ab78d22dc84f252bc545fbe6dd959f24c07c78d0d46d8c5672ebfb7b9f1eeeabf90afdc331f346f92c46547b40bc630114bdfd617314afd65ad60e3a6b83274aac7d6d7c586eba184f3e610f1f0d20bdd49c7cad481bac273915ad5d9cfcfe6c2e","0x8062cccf0a431a5b04e876d24d02d9d0a42e158ab5b37977d11b54967e9d74ff6020d0ee8b27e41c58a225cad60ad7a6d23e525b0fc3dff07a375f6c8636f89a107e1922292657a35d9cc3bec641f6da96df14e2875bfbdd8570660aa462b1b926003d92461b2b6ae0a3ebb1b682fa2c486270225ed5919813a7313a8db7433f","0x88af4dd2ac5a6ebf43755daba99dd125dd47a6099dcf4acb752fff1d497759e6564c4a6d69837f5ad79fcc4fcd8283c8ca604a9d34b10f25c730b9f8693cf42f3d67156811aa8fede71b7595cdc040fdbc1611c72e2ce35688a3c689919db60052e4503045e4d0c2ba729d9a7a5ff98f724d7a3410bb88f9f9d035b550281162"]}`

	robinGenesisJoinedGroup   = `{"GroupHash":"0x275d1ec7e591231e9de02a0d29cf465a5dfed5aea15feff938f85714bcc52296","GroupID":"0xd588982453c1f6278564b44667573ec3eb6ce326a63e33e89b091cfe3de2a887","GroupPK":"0x608fdd9a055d2e688401220c9e3d0e9c2f26c354b3be07f1a06c4c6d769ae3cd38828f2c7b5b531fff8306a61fbdfa94e5eb59cecff7b28f4abba21f6037a2792d36d3544b2d7057405482a2ebde48f06e302dd1c91fbb6a47d393bb3f19f5660e2909d46e8445f857f1e63dff2bd10913663ca3c8a6bcbc64b0d49d92693251","SignSecKey":{},"MemberSignPubkeyMap":{"0x2a17671c5a32175335fa098951ba50a9b4730aea7ecee86df6536297900f5b77":"0x81765735aa48fa38eab581a961630adb459bca8b33a540faed3abaf5baa0a4e44ca776e946ffe9b857421c21062ac3837684b72ef36d174d140f0990972f1da94509dba0edb65dd72b647143776b0d6c25c3a47d0419581d2730bae0e3c0ca34450cfb9346b7fadd6737ffb2d78048ec0360f2a57eb8db870d79d8e51ca31817","0x5437f9dd7171db9d04a8347dca5bf2b7789081631d79d2d7882c1774d2f4d123":"0x6a2a1db50572d6f026ab8274357e961229dbc0dc73e6e5e746fc08237875dffd725bb28a4e4e7e4b5fcc299ab3ff2fb18e64b1822a44b4486b2045d34b5b7e2d8a6a90742f02122be4efd09c6fb454ce606ee80666fd75f792c0e96140bc6b9283b90f32b5916bfa721e18422f5255a73ca9330151367bbc1731ff797a5c7496","0xb1979dd362353f0b59dff76cb223d5660a024db628257693f5470dec18c93160":"0x8cad31d6d8a33b3de6cf964c23aa66e17e56b5aa5914c9d3702871a2e2b41cb63c34a8741ac08ac8d64147b3b72b4ea26f59e6257d26ff59dc0e5088ea0a7e3188d20929c11b012b1e6ed2c2ef4e4b29b9ca6b4e1bd5adff6b899fba6e9df731837b85e47e53969b414839ee28efc4a245e7489726b9c9e5ff2d50fa1e676a4d"}}`
	mainNetGenesisJoinedGroup = `{"GroupHash":"0xaae596fd903c7f36ac02f6c97e5930a1ddb691795eefa98ae16f44089f0ad833","GroupID":"0xf8c4e4b03c091a76109065ce13035ff7e2bc7bfadb6ede8a9b9aa3927806c3e0","GroupPK":"0x2a7636f27fa9f38e70e4ce71134c00af9850d9788016e8aa694e2b107dfa75444b0c893af86a14b4d426c349ce933b863cc02e91f976999f7f8437b617af99aa1fa24ce78d93b6418f22048976722b6bf2a7310fcbf96ee6b8d138a0f00e79995187d043b6aa290548ccf62488bbaba9ddb64d78f913940ef71a97505fd8f6c2","SignSecKey":{},"MemberSignPubkeyMap":{"0x00e663f4e4f6ef09b0febaacf55e13b2408db978df935b908ef7027358e654f8":"0x6ef0476f3b330938b80c0d0f2a9d7c908a0b97be6330c6d066c274249aa7a05f8021fed4401ed3df95b283871b21fabc6f32f2a042f274da2cc9d9388750342e5bddbe3cd170d32a7195bee2cea4e0ce31a4c9ff6f17e81250302297c81a814303b2c54e90df84f18b2345ed254b43bd9330ebf5b47ed318be58d04f67f7b065","0x14398c018719479dd1ad124d1b991e4420c01d33c1e07c91f662b775f3e89f82":"0x60d5f518c935e70361881e000e74ea05591e76c997e8ff6537dde3b44a30d336612bcd9c4a9087aa7d406f4894faa75dfe762871268fd5dbe144f599148c0d9c02fd05d6ba69089e22af885371f561cf80db0359d200941832acdb4cd1e3f94b169da6e340b21eb4f22591fb12d3a83d3dccca5a02048b976926d9fce2372a64","0x1c55fb0fb9fa080f6885a14088f9efd54067f9de43fe7d05fb186c54cdd8090b":"0x6d4dd7b0cae6e5fcd14243e8df1b68288402fb7ed20dda2226261d22ac54865d63cf358db2d76844404b696ced2f4b9203bb2248f6c1cf13b57ced6b6b33d9c73d7b424b201a1a0612355cf6b32b8d73567747748d89bef41e63047cb1831a39123c7a8223635bb97bdddc8f8d082695cafb11887e6c230d14bca5cf1172efcf","0x22c4dd15964495f7d0c51b7c0d1af93f993c362fdd6d874df47fb4c220166267":"0x77dc2641a25da4b2adb10f805252079d1c60e3c6188816ef34c4686b37ca7f55537b0e8b2b07d82defa71b15129c9cb185dce962d3e05e7c10e05f2546a39e137a5af50f2ffe984f9062ccbc975eca5e1812b0e89a8dfd790d861b626eb92231087184ffcf13e005b47859d7cbb1f4b9141242cea21c8dd36c2a21ae565ba72f","0x4306a858b6e579fc2a54584ea13ebb474ead98e5269d2d8dfc9caf4d73acc772":"0x8b8d721971b6fcb9495fc36d979fdf6815d29e019662a88ac7a239830327b95c21dc745d3c77f62355865879339976466acf73d5948c855ca31bca05ebbdb21a04ffaff262db4ef4f8892380d9b544f8aa48d109ed2e0c118bd9792fd4c3cb60237682cb2e906ad50f132375ba08f426f6a7dd66426a8356afb191a7faf3688f","0x7b4a904a6c7aa9b75d1cfe18d93c421ec214b17037283a8e7690b5445836421d":"0x6bdc8ed7875c16665d6233226505b681e1c297b1d98c8ea4cab760c8711b1c49758be8f56cd3c7fee4cc03fe791e2c7a427fa0299745329bdde805803e3933d380b1bb48b85586e81c0147ee2e1d01205d9b4f278f14e41cfde9a76c000507d56ce2a897482728eba2ad060b974667a5004f4c03a7fd81381355ca130d78d81c","0x90d301d65727c2ab76abe5fd5020036e5b9065fb1d19022744c30a1ef5d4945a":"0x7c7930d41b24bd4319d598d009e601009d38e9f6b16712db87e12cd5047c1a3f2ef2af4903507b00900de6ad2c5b6d20b1fbb5425f38cfb188e0101fc8b39d551a4083aa8851d6f93a387490987b1b9779b80558dc47dd216e175cd3fb865bdc54f756737ff7f70b6ce76bfc6d4f3412f8b96dcb4f223de4872ba3b9c760ac9d","0x9b7ad7f53b230ba2919722497da8438898e3f574726e20230bd690e9e9d9d9fe":"0x7b4e0518253e6feac3da5543f7059eda960234e10173f860d5549418c70d078e66740dfbf20a11c760160fee95855b915c2efb42e96578ba1b48c4aa6c428faa0e9df114c1d886fe555f4f9ddff2f6cb7ff7e182e1ec37a292b5f78b8888891244549d123808290d3355c1970f904a49b39fa9370d17974b998c4b5aada5b547","0xbbd46b3e775c245343b2bfceeb9439bb824220726cbd9ed2229360496b663304":"0x282fa40d2a9b5f1ee3a2c2a49c393af08f84f7efb690d7e4e5e0638bc6ff0e57753d034279def455b1e4edf850ab170cbd6270a2ea7abae46ac2169739b3755c4edb5f5e3287c4a9b84c61383efdc52ff42d560a59b6e211f5c6da996b376bfd6a681e578181cbc54dea9885ab6d299acda841b55ab26dbe2397e5bec579e44e","0xe74d7ae7ab16efd47a4828ccf9e38732a724fb06f99f65bf41eb243167cd37d8":"0x2daec9c02c91ae8c3470b27db134d35bbdd5c8aa438dec10608020c6b1e3df896db1631d0634cb57d86eef9686a122a3ca78e7c405170c19ba4ee156f920a3ba4634ccd566c71051836e3aae9cc70b953a98d6714bfc1f2da7092e673cd7d6f7429ebad696db5e0f9c317d2cd04140fa461667e57609b785144b66ffad3bd028"}}&&{"GroupHash":"0xeec5ec8b6e74d9d03ff6e0cfbae37bebfee90580491616df795b22d887603ce1","GroupID":"0x1c0164fe4b56206321780ced0306f2b033b7d0140257b7d594aee0e1d7c4bb14","GroupPK":"0x6d22919147272e5b354f081dcca849a9785011cc3d67ac7a54c0002481ffcfdf56b777bb2a327e17d08f92b350253d99dfdbcf2ea4cc49850f6a4ac5a8205df43723f82057dbb10bd98d0b252e86b69d2eea60a167519f92f0326f73438d02c75d187defc44076fb48fff2c591ada4d4e94358ce9f65c0f05b56c94f73e89447","SignSecKey":{},"MemberSignPubkeyMap":{"0x138f0771a33357766ae663f4889e8b8be2654ac401214557c407868504051455":"0x0ef1a9a82fec1b81a108b6896d8f4bf794379dea7ed6f72342aa200d6f9d5c1a15cf924f1f8d4f255c3c161268b8fc771720b2bbf6219cdf39eb9afe1cfe579746f761eb9a6820fad587d9f69716dae2eded318e63a85e0daee0d67d27edc7710de2330ef3f49bb267a942c58e762eceb9a047029f460fed271b5ea67289a478","0x245f61a18215af0d2531b8e6921b13b52cd58fa43c2e19cd75c9031040ecc7fa":"0x781be1cc2ccf38a257c0473b4d2df100b5ebe7e599ab29de8f1c585b50f57c0320c9729acdf47205debec7b62ffa80c2ff114bf9a7a2be2bf6f8c2572b539a0186ea14678405a4c71e60f62a997b14d561755ec49cec57d16f01561b9d65afea720d7bd6bb29a63524ac7ef1940d448fe078e17b6a3a3c929a836209f2b3572a","0x3748f337f5915aaa3bc150fa22c7aebd1e3e397e3f61fb501eeacac1a268165e":"0x7479262eb3d78c3fc30f3a2e4da2ba88bca2d2ea8e7e851387885625038ed57315516fdc8fd2dd34b256eaafbc71d291cdc70f12d9d79089d3873eb46ec526dd2dcbe417f17fb4a0f233299abb5409e628e939975f2d9e26136fbd58fb6e5e7b629fb9ef9c6c622cdddc9556946f5dc9d96f6a33e51687295078772f73ef66bf","0x4d2300e3c25fd4214072e4adcf599abed366b5be1cea8de3829a80d48eca4205":"0x892772f126e6f0cce563f70a889a05dbe73a2e41460a2a8325a248543b8522d31b34c7b14862c3c2641f65d3c0eb188d82ece6694485a163b0b830ac856584fb6471cf46b73f937bafb18f9725d877ee400e6a4ddfb9ffd4a32e073ba7d913c01cf5e72a14937be16db84f597a8e13211a6c53b594daf833a34f85c70b74267c","0x6ddc3604fbb86696627828f7e9f33c85df6d66e19ecd12f29d7b095da32f3ed4":"0x6e051b3cb54a49ef575cb0fb0d09c11f5964aebd255b5d168e0a3ad063ed5e9e3f7178599abee2fdeb243bf2c53dafeb741516fe8b623c58fb6c255e27bd99c52067e4391251bca79f6956ced43fadffb8fe42a38e37fc7f134735d9e5c41494300b05ae8ed812e7825dca0efa898753a5814f7609b02f29407278f4149dfd50","0x77f144dcc7a25692a70ac7324504e764a3f6db0b3f954de87e8dc1a25467fe1a":"0x0d952d28e0120c051c69038712884ad039987c277a65d13ffb6a25a1325649677e4bde091ed18ff6c25b92f5508324de40b341b89a45db9348595466f6a1991a5a562657e4603950c073168a2d69e7aea937d03e8f06257fd82054e4438f9cb68e66acf6c3d23f5f49860b224ab6ee855829dd34db6b315398e19e5a46124bcd","0xab2d103f32136d9d635eeadfd82d6ac340788d3bf70862270ee04b82e94c79ee":"0x270dc84f34ec7c5d474770f1edbf3f3026faaeebfe9224cd560a9e84fc3e922306923f4e450b8e800d05055b779fa40f7bcff1b2d3bd5321485d9a5ea96b516c3dc8f5117c10b6f7f96a7b19bdaa9f5bc8675c5c60fcb6ba6da67e40e607ff4829518aa13ace030a518b7ec2648c87c2c5809f84fcb0d65cc548baedddc98c72","0xae003a027036ed72eac4d367fffd6979993b62f94e98a10c8d189ee462d19d84":"0x3646024b43ae604218148432fa4fd970703d7b984c31d616f1ec2a5982c7ac1b7ab373c2609755b27d3ec623367809d96f26808d2ea6029f6782b1f80791b49a805b6460f46eb9768ac36e7abc957bd2a390c80706a5c190bc5dca73c18458874e24269b46155b0edc92888925fb29d97c0beb1ee4db83e7840a1319cc1be6e9","0xbff465dd49992a79958dfa3c7e72a994558885c9d63f6ec5325701a74dde732f":"0x73a29e7035d0fef376ffa6fa91adb63d2e3a6552b3a155e4de893332c20b0536658f76fb84b60bd7b4a901df7b03a24d0cc1a90b96915db01d8b9717775c2129555db8c5680d1bfa13610057e7de85f2025e840fd936d5ab7e538b0d3d6f7b6449acb0f8e5b6a1963d045c321d2dbcfd808de4c10811251b04a853746252df5f","0xc8ad0be14a695432843e76485644eb41618c0307ceadd49fae57961f8ef716d1":"0x21ffd40d25c5ae71487d59b1e7321e555247ad35527064b1db904fde7020e6fa3b0669c35cc38f2c30131aace4a5406f8ef63da7a52102d7f5342f81323d48fe8857f4fcc064f0a56e31b67781be09c83ea4d1e83ad847a6071f4b70df3231ab38ec03afc62f490a11fc177a693dfcf81263604a9d3cffcbfcc671d31ace2f2a"}}`
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
