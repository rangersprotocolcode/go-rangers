pragma solidity ^0.4.18;

contract WRPG {
    string public name     = "";
    string public symbol   = "";
    uint8  public decimals = 18;
    uint256 private _totalSupply;

    event  Approval(address indexed src, address indexed guy, uint wad);
    event  Transfer(address indexed src, address indexed dst, uint wad);
    //event  Deposit(address indexed dst, uint wad);
    //event  Withdrawal(address indexed src, uint wad);

    mapping (address => uint)                       public  balanceOf;
    mapping (address => mapping (address => uint))  public  allowance;

    constructor(uint256 totalSupply,string _name,string _symbol) public {
        name = _name;
        symbol = _symbol;
        _totalSupply = totalSupply;
        //balanceOf[0xfe4d81a83def8ba75e5b9670c562d124cedb3e94] = _totalSupply-10000000000000000000;
        //balanceOf[0xf58e5Fab29788F914a38Ac710a36C950B7EBC9F3] = 10000000000000000000;
        //mainnet
        //balanceOf[0xF44Ce46191380AFB962460D3db9417d7d70E0Dd6] = 10000000000000000000;
    }

    function() public payable {
        //deposit();
    }

    function totalSupply() public view returns (uint) {
        return _totalSupply;
    }

    function approve(address guy, uint wad) public returns (bool) {
        allowance[msg.sender][guy] = wad;
        Approval(msg.sender, guy, wad);
        return true;
    }

    function transfer(address dst, uint wad) public returns (bool) {
        return transferFrom(msg.sender, dst, wad);
    }

    function transferFrom(address src, address dst, uint wad)
        public
        returns (bool)
    {
        require(balanceOf[src] >= wad);

        if (src != msg.sender && allowance[src][msg.sender] != uint(-1)) {
            require(allowance[src][msg.sender] >= wad);
            allowance[src][msg.sender] -= wad;
        }

        balanceOf[src] -= wad;
        balanceOf[dst] += wad;

        Transfer(src, dst, wad);

        return true;
    }
}