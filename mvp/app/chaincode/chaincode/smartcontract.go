package chaincode

import (
	"sort"
	"strings"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing a Rating
type SmartContract struct {
	contractapi.Contract
}

// Rating describes basic details of what makes up a simple rating
type Rating struct {
	Id      string `json:"Id"`
	Place   string `json:"Place"`
	Grade   int    `json:"Grade"`
	Comment string `json:"Comment"`
}

func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	ratings := []Rating{}

	for _, rating := range ratings {
		ratingJSON, err := json.Marshal(rating)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(rating.Id, ratingJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state: %v", err)
		}
	}

	return nil
}

func (s *SmartContract) CreateRating(ctx contractapi.TransactionContextInterface, id string, place string, grade int, comment string) error {
	exists, err := s.RatingExists(ctx, id)
	if err != nil {
		return fmt.Errorf("error checking rating existence: %v", err)
	}
	if exists {
		return fmt.Errorf("the rating %s already exists", id)
	}

	rating := Rating{
		Id:      id,
		Place:   place,
		Grade:   grade,
		Comment: comment,
	}
	ratingJSON, err := json.Marshal(rating)
	if err != nil {
		return fmt.Errorf("error marshalling rating JSON: %v", err)
	}

	if err := ctx.GetStub().PutState(id, ratingJSON); err != nil {
		return fmt.Errorf("error putting rating state: %v", err)
	}

	// Log successful
	ctx.GetStub().SetEvent("CreateRatingEvent", []byte(fmt.Sprintf("Rating %s created successfully", id)))

	return nil
}

func (s *SmartContract) GetAllRatings(ctx contractapi.TransactionContextInterface) ([]*Rating, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var ratings []*Rating
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var rating Rating
		err = json.Unmarshal(queryResponse.Value, &rating)
		if err != nil {
			return nil, err
		}
		ratings = append(ratings, &rating)
	}

	return ratings, nil
}

func (s *SmartContract) CalculateAverageGradeForPlace(ctx contractapi.TransactionContextInterface, place string) (float64, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return 0, err
	}
	defer resultsIterator.Close()

	var totalGrade, count int

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return 0, err
		}

		var rating Rating
		err = json.Unmarshal(queryResponse.Value, &rating)
		if err != nil {
			return 0, err
		}

		if rating.Place == place {
			totalGrade += rating.Grade
			count++
		}
	}

	if count == 0 {
		return 0, fmt.Errorf("no ratings found for place %s", place)
	}

	averageGrade := float64(totalGrade) / float64(count)
	return averageGrade, nil
}

func (s *SmartContract) GetRatingsForPlace(ctx contractapi.TransactionContextInterface, place string) ([]*Rating, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var ratingsForPlace []*Rating

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var rating Rating
		err = json.Unmarshal(queryResponse.Value, &rating)
		if err != nil {
			return nil, err
		}

		if rating.Place == place {
			ratingsForPlace = append(ratingsForPlace, &rating)
		}
	}

	if len(ratingsForPlace) == 0 {
		return nil, fmt.Errorf("no ratings found for place %s", place)
	}

	return ratingsForPlace, nil
}

func (s *SmartContract) GetRating(ctx contractapi.TransactionContextInterface) (string, error) {
    // Get all ratings
    allRatings, err := s.GetAllRatings(ctx)
    if err != nil {
        return "", fmt.Errorf("error getting all ratings: %v", err)
    }

    // Map to store the total grade and count for each place
    placeDataMap := make(map[string]struct {
        TotalGrade int
        Count      int
    })

    // Calculate total grade and count for each place
    for _, rating := range allRatings {
        data, exists := placeDataMap[rating.Place]
        if !exists {
            data = struct {
                TotalGrade int
                Count      int
            }{}
        }
        data.TotalGrade += rating.Grade
        data.Count++
        placeDataMap[rating.Place] = data
    }

    // Sort places by average grade
    var sortedPlaces []string
    for place := range placeDataMap {
        sortedPlaces = append(sortedPlaces, place)
    }

    sort.Slice(sortedPlaces, func(i, j int) bool {
        placeI, placeJ := sortedPlaces[i], sortedPlaces[j]
        averageGradeI := float64(placeDataMap[placeI].TotalGrade) / float64(placeDataMap[placeI].Count)
        averageGradeJ := float64(placeDataMap[placeJ].TotalGrade) / float64(placeDataMap[placeJ].Count)
        return averageGradeI > averageGradeJ
    })

    // Build the output string
    var outputString strings.Builder
    outputString.WriteString("Average Grade\t\tPlace\n")
    for _, place := range sortedPlaces {
        averageGrade := float64(placeDataMap[place].TotalGrade) / float64(placeDataMap[place].Count)
        outputString.WriteString(fmt.Sprintf("%.2f\t\t%s\n", averageGrade, place))
    }

    // Return the output string directly
    return outputString.String(), nil
}

func (s *SmartContract) AreRatingsEmpty(ctx contractapi.TransactionContextInterface) (bool, error) {
    resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
    if err != nil {
        return false, err
    }
    defer resultsIterator.Close()

    return !resultsIterator.HasNext(), nil
}

func (s *SmartContract) ReadRating(ctx contractapi.TransactionContextInterface, id string) (*Rating, error) {
	ratingJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if ratingJSON == nil {
		return nil, fmt.Errorf("the rating %s does not exist", id)
	}

	var rating Rating
	err = json.Unmarshal(ratingJSON, &rating)
	if err != nil {
		return nil, err
	}

	return &rating, nil
}

func (s *SmartContract) UpdateRating(ctx contractapi.TransactionContextInterface, id string, place string, grade int, comment string) error {
	exists, err := s.RatingExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the rating %s does not exist", id)
	}

	// overwriting original rating with new rating
	rating := Rating{
		Id:      id,
		Place:   place,
		Grade:   grade,
		Comment: comment,
	}
	ratingJSON, err := json.Marshal(rating)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, ratingJSON)
}

func (s *SmartContract) DeleteRating(ctx contractapi.TransactionContextInterface, id string) error {
	exists, err := s.RatingExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the rating %s does not exist", id)
	}

	return ctx.GetStub().DelState(id)
}

func (s *SmartContract) RatingExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	ratingJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return ratingJSON != nil, nil
}
