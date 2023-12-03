# Create the test network and a channel (from the test-network folder).
cd test-network 
./network.sh up createChannel -c mychannel -ca

# Deploy one of the smart contract implementations.
./network.sh deployCC -ccn basic -ccp ../app/chaincode/ -ccl go

# Run
cd ../app/application
go run .

# Clean up
cd test-network 
./network.sh down

peer lifecycle chaincode querycommitted --channelID mychannel --name basic
