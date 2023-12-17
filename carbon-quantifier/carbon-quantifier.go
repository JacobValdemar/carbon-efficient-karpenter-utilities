package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type InstanceList struct {
	Instances []Instance
	Location  string
}

type Instance struct {
	APIName    string
	GWPperHour float64
}

const BoaviztAPIBaseURL string = "http://localhost:5001" // or https://api.boavizta.org
const OutputLocation string = "zz_generated.carbon.go"

func main() {
	instances := getAllInstances()
	instances = filterInstances(instances)

	// TODO: Use all regions in https://github.com/vantage-sh/ec2instances.info/blob/master/meta/regions_aws.yaml
	var locationToRegion map[string][]string = map[string][]string{
		"USA": {"us-east-1", "us-east-2", "us-west-1", "us-west-2"},
		"ZAF": {"af-south-1"},
		"HKG": {"ap-east-1"},
		"IND": {"ap-south-1", "ap-south-2"},
		"JPN": {"ap-northeast-1", "ap-northeast-3"},
		"KOR": {"ap-northeast-2"},
		"SGP": {"ap-southeast-1"},
		"AUS": {"ap-southeast-2", "ap-southeast-4"},
		"IDN": {"ap-southeast-3"},
		"CAN": {"ca-central-1"},
		"DEU": {"eu-central-1"},
		"CHE": {"eu-central-2"},
		"IRL": {"eu-west-1"},
		"GBR": {"eu-west-2"},
		"FRA": {"eu-west-3"},
		"ITA": {"eu-south-1"},
		"ESP": {"eu-south-2"},
		"SWE": {"eu-north-1"},
		"ISR": {"il-central-1"},
		"BHR": {"me-south-1"},
		"ARE": {"me-central-1"},
		"BRA": {"sa-east-1"},
	}

	var result []InstanceList

	for location := range locationToRegion {
		fmt.Printf("%s\n", location)
		var instancesInThisLocation []Instance
		for _, instance := range instances {
			embodied, operational, _, _ := getEmissions(instance, location)
			instancesInThisLocation = append(instancesInThisLocation, Instance{APIName: instance, GWPperHour: embodied + operational})
		}
		sort.Slice(instancesInThisLocation, func(i, j int) bool {
			return instancesInThisLocation[i].APIName < instancesInThisLocation[j].APIName
		})
		result = append(result, InstanceList{
			Location:  location,
			Instances: instancesInThisLocation,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Location < result[j].Location
	})

	writeToGoFile(OutputLocation, result, locationToRegion)
}

func writeToGoFile(fileName string, instanceList []InstanceList, locationToRegion map[string][]string) {
	now := time.Now().UTC().Format(time.RFC3339)
	var stringToWrite []byte
	stringToWrite = fmt.Appendf(stringToWrite, "//go:build !ignore_autogenerated\n\n")
	stringToWrite = fmt.Appendf(stringToWrite, "package carbon\n\nimport \"time\"\n\n// generated at %s\n\nvar initialPriceUpdate, _ = time.Parse(time.RFC3339, \"%s\")\nvar carbonImpacts = map[string]*map[string]float64{}\n\nfunc init() {", now, now)

	for _, instanceRegion := range instanceList {
		stringToWrite = fmt.Appendf(stringToWrite, "\n    carbonImpacts[\"%s\"] = &map[string]float64{", instanceRegion.Location)
		for _, priceOverride := range instanceRegion.Instances {
			stringToWrite = fmt.Appendf(stringToWrite, "\n        \"%s\": %f,", priceOverride.APIName, priceOverride.GWPperHour)
		}
		stringToWrite = fmt.Appendf(stringToWrite, "\n    }\n")
	}

	for location, regions := range locationToRegion {
		for _, region := range regions {
			stringToWrite = fmt.Appendf(stringToWrite, "\n    carbonImpacts[\"%s\"] = carbonImpacts[\"%s\"]", region, location)
		}
	}

	stringToWrite = fmt.Appendf(stringToWrite, "\n}\n")

	err := os.WriteFile(fileName, stringToWrite, 0644)
	if err != nil {
		panic(err)
	}
}

func getEmissions(instanceType string, usageLocation string) (float64, float64, float64, float64) {
	url := BoaviztAPIBaseURL + "/v1/cloud/instance?verbose=true&duration=1&criteria=gwp"

	payload := []byte(`{
		"provider": "aws",
		"instance_type": "` + instanceType + `",
		"usage": {
		  "usage_location": "` + usageLocation + `",
		  "time_workload": [
			{
			  "time_percentage": 100,
			  "load_percentage": 100
			}
		  ]
		}
	  }`)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var response ResponseGenerated
	err = json.Unmarshal([]byte(body), &response)
	if err != nil {
		panic(err)
	}

	return response.Impacts.Gwp.Embedded.Value,
		response.Impacts.Gwp.Use.Value,
		(response.Verbose.CPU1.CoreUnits.Value * response.Verbose.CPU1.Units.Value) / response.Verbose.InstancePerServer.Value,
		(response.Verbose.RAM1.Capacity.Value * response.Verbose.RAM1.Units.Value) / response.Verbose.InstancePerServer.Value
}

func getAllInstances() []string {
	url := BoaviztAPIBaseURL + "/v1/cloud/instance/all_instances?provider=aws"

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var response []string
	err = json.Unmarshal([]byte(body), &response)
	if err != nil {
		panic(err)
	}

	return response
}

func filterInstances(instances []string) []string {
	var newInstances []string

	for _, v := range instances {
		if strings.Contains(v, ".elasticsearch") {
			continue
		}
		if strings.Contains(v, "cache.") {
			continue
		}
		if strings.Contains(v, "db.") {
			continue
		}
		newInstances = append(newInstances, v)
	}

	return newInstances
}

type ResponseGenerated struct {
	Impacts struct {
		Gwp struct {
			Embedded struct {
				Value              float64  `json:"value,omitempty"`
				SignificantFigures int      `json:"significant_figures,omitempty"`
				Min                float64  `json:"min,omitempty"`
				Max                float64  `json:"max,omitempty"`
				Warnings           []string `json:"warnings,omitempty"`
			} `json:"embedded,omitempty"`
			Use struct {
				Value              float64 `json:"value,omitempty"`
				SignificantFigures int     `json:"significant_figures,omitempty"`
				Min                float64 `json:"min,omitempty"`
				Max                float64 `json:"max,omitempty"`
			} `json:"use,omitempty"`
			Unit        string `json:"unit,omitempty"`
			Description string `json:"description,omitempty"`
		} `json:"gwp,omitempty"`
	} `json:"impacts,omitempty"`
	Verbose struct {
		CPU1 struct {
			CoreUnits struct {
				Value float64 `json:"value,omitempty"`
			} `json:"core_units,omitempty"`
			Units struct {
				Value float64 `json:"value,omitempty"`
			} `json:"units,omitempty"`
		} `json:"CPU-1,omitempty"`
		RAM1 struct {
			Capacity struct {
				Value float64 `json:"value,omitempty"`
			} `json:"capacity,omitempty"`
			Units struct {
				Value float64 `json:"value,omitempty"`
			} `json:"units,omitempty"`
		} `json:"RAM-1,omitempty"`
		InstancePerServer struct {
			Value float64 `json:"value,omitempty"`
		} `json:"instance_per_server,omitempty"`
	} `json:"verbose,omitempty"`
}
