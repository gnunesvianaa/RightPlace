pragma solidity ^0.8.22;

contract MyContract {

    mapping(uint => Person) public people;

    uint256 public peopleCount = 0;

    uint256 public opennigTime = 1701128199;

    modifier onlyWhileOpen() {
        require(block.timestamp >= opennigTime);
        _;
    }

    struct Person{
        uint _id;
        string _firstName;
        string _lastName;
    }

    function addPerson(
        string memory _firstName, 
        string memory _lastName
    ) 
        public 
        onlyWhileOpen
    {
        incrementConut();
        people[peopleCount] = Person(peopleCount, _firstName, _lastName);
    }

    function incrementConut() internal{
        peopleCount += 1;
    }

}
