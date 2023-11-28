// SPDX-License-Identifier: MIT
// safe math is good lbrary
pragma solidity ^0.8.22;

library Math {
    function divide(uint256 a, uint256 b) internal pure returns(uint256){
        require(a>0);
        uint256 c = a/b;
        return c;
    }
}

contract MyContract{
    uint256 public value;

    function calculate(uint _v1, uint _v2) public{
        
        value = Math.divide(_v1, _v2);
    }
}
