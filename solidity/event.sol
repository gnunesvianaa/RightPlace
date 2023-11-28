pragma solidity ^0.8.22;

contract MyContract {

    mapping(address => uint256) public balances;
    address payable wallet;

    event Purchase(
        address indexed _buyer,
        uint _amount
    );

    constructor(address payable  _wallet){
        wallet = _wallet;
    }

    // fallback() external payable {
    //     buyToken();
    // }

    function buyToken() public payable {  // Add payable modifier
        // buy a token
        balances[msg.sender] += 1;
        // send ether to the wallet
        wallet.transfer(msg.value);
        emit Purchase(msg.sender, 1);
    }

}
