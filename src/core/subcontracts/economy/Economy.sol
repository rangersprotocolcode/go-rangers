// SPDX-License-Identifier: MIT
pragma solidity 0.7.5;

import  "./proxy/TransparentUpgradeableProxy.sol";

/**
 * @title SafeMath
 * @dev Unsigned math operations with safety checks that revert on error.
 */
library SafeMath {
    /**
     * @dev Multiplies two unsigned integers, reverts on overflow.
     */
    function mul(uint256 a, uint256 b) internal pure returns (uint256) {
        // Gas optimization: this is cheaper than requiring 'a' not being zero, but the
        // benefit is lost if 'b' is also tested.
        // See: https://github.com/OpenZeppelin/openzeppelin-solidity/pull/522
        if (a == 0) {
            return 0;
        }

        uint256 c = a * b;
        require(c / a == b);

        return c;
    }

    /**
     * @dev Integer division of two unsigned integers truncating the quotient, reverts on division by zero.
     */
    function div(uint256 a, uint256 b) internal pure returns (uint256) {
        // Solidity only automatically asserts when dividing by 0
        require(b > 0);
        uint256 c = a / b;
        // assert(a == b * c + a % b); // There is no case in which this doesn't hold

        return c;
    }

    /**
     * @dev Subtracts two unsigned integers, reverts on overflow (i.e. if subtrahend is greater than minuend).
     */
    function sub(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b <= a);
        uint256 c = a - b;

        return c;
    }

    /**
     * @dev Adds two unsigned integers, reverts on overflow.
     */
    function add(uint256 a, uint256 b) internal pure returns (uint256) {
        uint256 c = a + b;
        require(c >= a);

        return c;
    }

    /**
     * @dev Divides two unsigned integers and returns the remainder (unsigned integer modulo),
     * reverts when dividing by zero.
     */
    function mod(uint256 a, uint256 b) internal pure returns (uint256) {
        require(b != 0);
        return a % b;
    }
}

contract Economy{
    using SafeMath for uint256;
    uint256 public  decimal = 18;
    uint256 public  total;  // ¹©Ó¦×ÜÁ¿
    uint256 public  epoch;  // Í¨ËõÖÜÆÚ¡£¼ÙÉèÍ¨ËõÖÜÆÚÊÇ180Ìì£¬1Ãë1¿é£¬ÔòÕâÀïµÄÖµÎª180*24*3600/1
    uint256 public  rate;   // Í¨ËõÂÊ 8´ú±í8%

    uint256 public  proposalRate;       // ·Ö¸ø³ö¿éÕßµÄÕ¼±ÈÊý 30´ú±í30%
    uint256 public  otherProposalRate;  // ·Ö¸øËùÓÐÌá°¸ÕßµÄÕ¼±È 35´ú±í35%



    constructor(uint256 _total, uint256 _epoch, uint256 _rate, uint256 _proposalRate, uint256 _otherProposalRate) {
       total = _total;
       epoch = _epoch;
       rate = _rate;

       proposalRate = _proposalRate;
       otherProposalRate = _otherProposalRate;
    }



    function calc(address payable proposal, address payable[] calldata allProposal, address payable[] calldata validators) external {
        require(msg.sender==address(0x1111111111111111111111111111111111111111),'calc caller wrong');

        uint256 reward = calcPerBlock(block.number);


        proposal.transfer(reward.mul(proposalRate).div(100));


        uint256 otherProposal = reward.mul(otherProposalRate).div(100);

        uint256 totalStake = 0;
        for(uint256 i=0;i<allProposal.length;i++){
            totalStake = totalStake.add(allProposal[i].stakenum);
        }
        for(uint256 i=0;i<allProposal.length;i++){
            //allProposal[i].balance = allProposal[i].balance+otherProposal * allProposal[i].stakenum/totalStake
            allProposal[i].transfer(otherProposal.mul(allProposal[i].stakenum).div(totalStake));
        }


        uint256 validatorReward = reward.mul(uint256(100).sub(otherProposalRate).sub(proposalRate)).div(100); //ÕâÑù¼ÆËã¿ÉÄÜÓÐÒ»µãµãÎó²î
        uint256 totalValidatorStake = 0;
        for(uint256 i=0;i<validators.length;i++){
            totalValidatorStake = totalValidatorStake.add(validators[i].stakenum);
        }
        for(uint256 i=0;i<validators.length;i++){
            //validators[i].balance = validators[i].balance+validatorReward * validators[i].stakenum/totalValidatorStake
            validators[i].transfer(validatorReward.mul(validators[i].stakenum).div(totalValidatorStake));

        }
    }

    function calcPerBlock(uint256 height) /*internal*/public view returns(uint256){
        //total * math.Pow(1-rate, float64(getEpoch(height))) * rate / epoch

        uint256 _epoch = getEpoch(height);
        uint256 thisall = total;
        for(uint256 i = 0 ; i < _epoch; i++){
            thisall = thisall.mul(100-rate).div(100);
        }

        return thisall.div(epoch).mul(rate).div(100);
    }

    function getEpoch(uint256 height) internal view returns(uint256) {
        return height.div(epoch);
    }
}

contract EcomonyProxy is TransparentUpgradeableProxy {
    constructor(address logic, address admin, bytes memory data) payable TransparentUpgradeableProxy(logic, admin, data) {}
}