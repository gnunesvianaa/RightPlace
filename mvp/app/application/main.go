package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"log"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"
	"strings"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

type Rating struct {
	Id      string    `json:"id"`
	Place   string `json:"place"`
	Grade   int    `json:"grade"`
	Comment string `json:"comment"`
}

const (
	mspID        = "Org1MSP"
	cryptoPath   = "../../network/organizations/peerOrganizations/org1.example.com"
	certPath     = cryptoPath + "/users/User1@org1.example.com/msp/signcerts/cert.pem"
	keyPath      = cryptoPath + "/users/User1@org1.example.com/msp/keystore/"
	tlsCertPath  = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint = "localhost:7051"
	gatewayPeer  = "peer0.org1.example.com"
)

var now = time.Now()
var assetId = fmt.Sprintf("asset%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)

func main() {
	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Override default values for chaincode and channel name as they may differ in testing contexts.
	chaincodeName := "basic"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname
	}

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	// code
	initLedger(contract)

	for {
		fmt.Println("\nChoose an option:")
		fmt.Println("1. Add an Rating")
		fmt.Println("2. View all Ratings")
		fmt.Println("3. Search Place")
		fmt.Println("4. Overall Rating")
        fmt.Println("5. Exit")

		var choice int
		fmt.Print("Enter your choice: ")
		_, err := fmt.Scan(&choice)
		if err != nil {
			fmt.Println("Invalid input. Please enter a number.")
			continue
		}
		fmt.Println("\n")

		switch choice {
		case 1:
			addRating(contract)
		case 2:
			getAllRatings(contract)
		case 3:
			searchPlace(contract)
		case 4:
			viewRating(contract)
		case 5:
			fmt.Println("Exiting the program.")
			return
		default:
			fmt.Println("Invalid choice. Please enter a valid option.")
		}
	}
}

// initial deployment. A new version of the chaincode deployed later would likely not need to run an "init" function.
func initLedger(contract *client.Contract) {
	fmt.Printf("\n--> Submit Transaction: InitLedger, function creates the initial set of ratings on the ledger \n")

	_, err := contract.SubmitTransaction("InitLedger")
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

func addRating(contract *client.Contract) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter the place: ")
	scanner.Scan()
	place := scanner.Text()

	fmt.Print("Enter the grade: ")
	scanner.Scan()
	gradeStr := scanner.Text()
	grade, err := strconv.Atoi(gradeStr)
	if err != nil {
		fmt.Println("Invalid grade. Please enter a valid number.")
		return
	}

	fmt.Print("Enter the comment: ")
	scanner.Scan()
	comment := scanner.Text()

	createRating(contract, place, grade, comment)
	fmt.Println("Rating added successfully.")
}

func createRating(contract *client.Contract, place string, grade int, comment string) {
	fmt.Printf("\n--> Submit Transaction: CreateRating, creates new rating with ID, Place, Grade, Comment arguments \n")

	// Gere um novo ID único para cada rating
	assetId := fmt.Sprintf("asset%d", time.Now().UnixNano())

	_, err := contract.SubmitTransaction("CreateRating", assetId, place, strconv.Itoa(grade), comment)
	if err != nil {
		fmt.Printf("Failed to submit transaction: %v\n", err)
		return
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

func getAllRatings(contract *client.Contract) {
    fmt.Println("\n--> Evaluate Transaction: GetAllRatings, function returns all the current ratings on the ledger")

    evaluateResult, err := contract.EvaluateTransaction("GetAllRatings")
    if err != nil {
        panic(fmt.Errorf("failed to evaluate transaction: %w", err))
    }

    // Check if there are no ratings
    if len(evaluateResult) == 0 {
        fmt.Println("No ratings found.")
        return
    }

    var ratings []*Rating
    err = json.Unmarshal(evaluateResult, &ratings)
    if err != nil {
        panic(fmt.Errorf("failed to unmarshal JSON: %w", err))
    }

    // Display the results in a more user-friendly format
    fmt.Println("*** Ratings:")
    for _, rating := range ratings {
        fmt.Printf("ID: %s\n", rating.Id)
        fmt.Printf("Place: %s\n", rating.Place)
        fmt.Printf("Grade: %d\n", rating.Grade)
        fmt.Printf("Comment: %s\n", rating.Comment)
        fmt.Println("----")
    }
}

func searchPlace(contract *client.Contract) {
    scanner := bufio.NewScanner(os.Stdin)

    fmt.Print("Enter the place to search: ")
    scanner.Scan()
    place := scanner.Text()

    results, err := contract.EvaluateTransaction("GetRatingsForPlace", place)
    if err != nil {
        if strings.Contains(err.Error(), "no ratings found for place") {
            fmt.Printf("No ratings found for place %s\n", place)
            return
        }
        log.Fatal(err)
    }

    // Assuming results is a JSON array of ratings
    var ratingsForPlace []*Rating
    err = json.Unmarshal(results, &ratingsForPlace)
    if err != nil {
        log.Fatal(err)
    }

    if len(ratingsForPlace) == 0 {
        fmt.Printf("No ratings found for place %s\n", place)
        return
    }

    var totalGrade, count int

    for _, rating := range ratingsForPlace {
        totalGrade += rating.Grade
        count++
    }

    averageGrade := float64(totalGrade) / float64(count)

    // Display the results
    fmt.Printf("Average Grade for %s: %.2f\n", place, averageGrade)
    fmt.Printf("Ratings for %s:\n", place)
    for _, rating := range ratingsForPlace {
        fmt.Printf("Grade: %d, Comment: %s\n",rating.Grade, rating.Comment)
    }
}

func viewRating(contract *client.Contract) {
    fmt.Println("\n--> Checking if ratings are empty...")

    isEmpty, err := contract.EvaluateTransaction("AreRatingsEmpty")
    if err != nil {
        panic(fmt.Errorf("failed to evaluate transaction for checking if ratings are empty: %w", err))
    }

    // Check if ratings are empty
    if strings.Contains(strings.ToLower(string(isEmpty)), "true") {
        // No ratings found
        fmt.Println("No ratings found. Please add ratings to view the average grades.")
    } else {
        // Ratings are present, proceed with GetRating
        fmt.Println("\n--> Evaluate Transaction: GetRating, function returns a sorted table with average grades for each place")

        evaluateResult, err := contract.EvaluateTransaction("GetRating")
        if err != nil {
            panic(fmt.Errorf("failed to evaluate transaction: %w", err))
        }

        // Print the raw response directly
        fmt.Printf("%s\n", evaluateResult)
    }
}

func newGrpcConnection() *grpc.ClientConn {
	certificate, err := loadCertificate(tlsCertPath)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.Dial(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

func newIdentity() *identity.X509Identity {
	certificate, err := loadCertificate(certPath)
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}

	return id
}

func loadCertificate(filename string) (*x509.Certificate, error) {
	certificatePEM, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}
	return identity.CertificateFromPEM(certificatePEM)
}

func newSign() identity.Sign {
	files, err := os.ReadDir(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key directory: %w", err))
	}
	privateKeyPEM, err := os.ReadFile(path.Join(keyPath, files[0].Name()))

	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}

	return sign
}

func readAssetByID(contract *client.Contract) {
	fmt.Printf("\n--> Evaluate Transaction: ReadAsset, function returns asset attributes\n")

	evaluateResult, err := contract.EvaluateTransaction("ReadAsset", assetId)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

func transferAssetAsync(contract *client.Contract) {
	fmt.Printf("\n--> Async Submit Transaction: TransferAsset, updates existing asset owner")

	submitResult, commit, err := contract.SubmitAsync("TransferAsset", client.WithArguments(assetId, "Mark"))
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction asynchronously: %w", err))
	}

	fmt.Printf("\n*** Successfully submitted transaction to transfer ownership from %s to Mark. \n", string(submitResult))
	fmt.Println("*** Waiting for transaction commit.")

	if commitStatus, err := commit.Status(); err != nil {
		panic(fmt.Errorf("failed to get commit status: %w", err))
	} else if !commitStatus.Successful {
		panic(fmt.Errorf("transaction %s failed to commit with status: %d", commitStatus.TransactionID, int32(commitStatus.Code)))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

func exampleErrorHandling(contract *client.Contract) {
	fmt.Println("\n--> Submit Transaction: UpdateAsset asset70, asset70 does not exist and should return an error")

	_, err := contract.SubmitTransaction("UpdateAsset", "asset70", "blue", "5", "Tomoko", "300")
	if err == nil {
		panic("******** FAILED to return an error")
	}

	fmt.Println("*** Successfully caught the error:")

	switch err := err.(type) {
	case *client.EndorseError:
		fmt.Printf("Endorse error for transaction %s with gRPC status %v: %s\n", err.TransactionID, status.Code(err), err)
	case *client.SubmitError:
		fmt.Printf("Submit error for transaction %s with gRPC status %v: %s\n", err.TransactionID, status.Code(err), err)
	case *client.CommitStatusError:
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("Timeout waiting for transaction %s commit status: %s", err.TransactionID, err)
		} else {
			fmt.Printf("Error obtaining commit status for transaction %s with gRPC status %v: %s\n", err.TransactionID, status.Code(err), err)
		}
	case *client.CommitError:
		fmt.Printf("Transaction %s failed to commit with status %d: %s\n", err.TransactionID, int32(err.Code), err)
	default:
		panic(fmt.Errorf("unexpected error type %T: %w", err, err))
	}

	// Any error that originates from a peer or orderer node external to the gateway will have its details
	// embedded within the gRPC status error. The following code shows how to extract that.
	statusErr := status.Convert(err)

	details := statusErr.Details()
	if len(details) > 0 {
		fmt.Println("Error Details:")

		for _, detail := range details {
			switch detail := detail.(type) {
			case *gateway.ErrorDetail:
				fmt.Printf("- address: %s, mspId: %s, message: %s\n", detail.Address, detail.MspId, detail.Message)
			}
		}
	}
}

func formatJSON(data []byte) string {
    var prettyJSON bytes.Buffer
    if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
        panic(fmt.Errorf("failed to parse JSON: %w", err))
    }
    return prettyJSON.String()
}