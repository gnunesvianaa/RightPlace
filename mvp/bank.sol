pragma solidity 0.8.23;

contract Bank {

    address payable owner;

    mapping (address => uint) public balance;

    event depositLog(address account, uint value);
    event withdrawalLog(address account, uint value);


    constructor() payable {
        owner = payable(msg.sender);
    }

    function withdrawal(uint256 _value) external{
        require(msg.sender == owner, "caller is not owner");
        balance[owner] -= _value;
        emit withdrawalLog(address(this), _value);
    }

    function deposit(address payable receiver, uint _value) external payable {
        balance[msg.sender] = balance[msg.sender] + msg.value;
        //receiver.transfer(_value);

        emit depositLog(address(this), _value);
    }

    function getBalance() external view returns (uint){
        //return address(this).balance;
        return balance[msg.sender];
    }
}