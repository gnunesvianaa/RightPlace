# Create the test network and a channel.
cd network 
./network.sh up createChannel -c mychannel -ca

# Deploy one of the smart contract implementations.
./network.sh deployCC -ccn basic -ccp ../app/chaincode/ -ccl go

# Run
cd ../app/application
go run .

# Clean up
cd network 
./network.sh down