pragma solidity ^0.8.22;

contract MyContract {

    enum State {wait, active, stop}

    State public state;

    constructor() {
        state = State.wait;
    }

    function activate() public{
        state = State.active;
    }

    function isActive() public view returns(bool){
        return state == State.active;
    }

}
