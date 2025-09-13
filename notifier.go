package main

import (
	"os"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"strings"
	"strconv"
)

type Tracker struct {
	SeenUUID map[string]struct{}
	TaxonIDs []int
	PlaceID int
	NtfyURL string
	NtfyToken string
}

func (t *Tracker) Seen(uuid string) bool {
	if _, ok := t.SeenUUID[uuid]; ok {
		return true
	}
	return false
}

func (t *Tracker) AddID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	t.TaxonIDs = append(t.TaxonIDs, id)
	return nil
}

func NewTracker() Tracker {
	t := Tracker{}
	t.SeenUUID = make(map[string]struct{})
	return t
}

func (t *Tracker) SendNotification(titleStr string, bodyStr string, observationID int) {
	body := bytes.NewBufferString(bodyStr)

	req, err := http.NewRequest("POST", t.NtfyURL, body)
	if err != nil {
		fmt.Printf("error creating request %v\n", err)
	}

	req.Header.Set("Title", titleStr)
	req.Header.Set("Actions", fmt.Sprintf("view, Open Observation, https://www.inaturalist.org/observations/%d, clear=true", observationID))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.NtfyToken))
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("error sending notification: %v\n", err)
	}
	if resp.StatusCode != 200 {
		fmt.Printf("error sending notification: %d\n", resp.StatusCode)
	}
	defer resp.Body.Close()
}

type APIResponse struct {
	TotalResults int `json:"total_results"`
	Page         int `json:"page"`
	PerPage      int `json:"per_page"`
	Results      []APIResponseResult `json:"results"`
}

type APIResponseResult struct {
		UUID             string    `json:"uuid"`
		CreatedAt        time.Time `json:"created_at"`
		CreatedAtDetails struct {
			Date  string `json:"date"`
			Day   int    `json:"day"`
			Hour  int    `json:"hour"`
			Month int    `json:"month"`
			Week  int    `json:"week"`
			Year  int    `json:"year"`
		} `json:"created_at_details"`
		CreatedTimeZone   string `json:"created_time_zone"`
		Geoprivacy        any    `json:"geoprivacy"`
		ID                int    `json:"id"`
		Location          string `json:"location"`
		Mappable          bool   `json:"mappable"`
		Obscured          bool   `json:"obscured"`
		ObservedOn        string `json:"observed_on"`
		ObservedOnDetails struct {
			Date  string `json:"date"`
			Day   int    `json:"day"`
			Hour  int    `json:"hour"`
			Month int    `json:"month"`
			Week  int    `json:"week"`
			Year  int    `json:"year"`
		} `json:"observed_on_details"`
		ObservedTimeZone string `json:"observed_time_zone"`
		PlaceGuess       string `json:"place_guess"`
		QualityGrade     string `json:"quality_grade"`
		Taxon struct {
			ID int `json:"id"`
			PreferredCommonName string `json:"preferred_common_name"`
		} `json:"taxon"`
}

func (r *APIResponseResult) ObscuredAsString() string {
	if r.Obscured {
		return "yes"
	}
	return "no"
}

func GetObservation(taxon int, place int) (*APIResponse, error) {
	req, err := http.NewRequest("GET", "https://api.inaturalist.org/v2/observations", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	q := req.URL.Query()
	q.Add("verifiable", "true")
	q.Add("order_by", "created_at")
	q.Add("order", "desc")
	q.Add("page", "1")
	q.Add("spam", "false")
	q.Add("taxon_id", fmt.Sprintf("%d", taxon))
	q.Add("place_id", fmt.Sprintf("%d", place))
	q.Add("locale", "en-US")
	q.Add("per_page", "50")
	q.Add("fields", "(created_at:!t,created_at_details:all,created_time_zone:!t,geoprivacy:!t,id:!t,location:!t,mappable:!t,obscured:!t,observed_on:!t,observed_on_details:all,observed_time_zone:!t,place_guess:!t,private_geojson:!t,quality_grade:!t,taxon:(preferred_common_name:!t))")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("error getting observations: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error getting observations: status code is not 200 ( %d )", resp.StatusCode)
	}

	// Read JSON response
	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading observations json: %v", err)
	}

	// fmt.Printf("%s\n", rbody)

	// Convert JSON response to APIResponse
	r := APIResponse{}
	err = json.Unmarshal(rbody, &r)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling observations: %v", err)
	}

	return &r, nil
}

func main() {
	firstRun := true
	taxonIDs := os.Getenv("TAXON_IDS")
	placeIDStr := os.Getenv("PLACE_ID")
	tracker := NewTracker()
	tracker.NtfyToken = os.Getenv("NTFY_TOKEN")
	tracker.NtfyURL = os.Getenv("NTFY_URL")

	if tracker.NtfyToken == "" {
		fmt.Printf("ERROR: NTFY_TOKEN needs to be set.\n")
		return
	}
	if tracker.NtfyURL == "" {
		fmt.Printf("ERROR: NTFY_URL should be the URL to your Ntfy sub.\n")
		return
	}
	if placeIDStr == "" {
		fmt.Printf("ERROR: PLACE_ID should be the URL to your Ntfy sub.\n")
		return
	}
	if taxonIDs == "" {
		fmt.Printf("ERROR: TAXON_IDS should be a list of taxon IDs, separated by commas. No spaces.\n")
		return
	}

	var err error
	tracker.PlaceID, err = strconv.Atoi(placeIDStr)
	if err != nil {
		fmt.Printf("ERROR: Unable to convert place ID to integer\n")
		return
	}

	// Add taxon IDs to tracker
	for _, tid := range strings.Split(taxonIDs, ",") {
		tracker.AddID(tid)
	}

	for {
		// Create a list to store the notification UUIDs for the run
		thisRunUUID := make(map[string]struct{})
		for _, taxon := range tracker.TaxonIDs {

			// Retrieve observations for the taxon ID
			fmt.Printf("Getting observations for taxon ID '%d'\n", taxon)
			r, err := GetObservation(taxon, tracker.PlaceID)
			if err != nil {
				fmt.Printf("%+v", err)
			}

			if len(r.Results) > 0 {
				fmt.Printf("Taxon ID '%d' is the '%s'\n", taxon, r.Results[0].Taxon.PreferredCommonName)
			} else {
				fmt.Printf("No results found for Taxon ID '%d'\n", taxon)
				continue
			}

			// Iterate the results
			for _, o := range r.Results {
				if !firstRun && !tracker.Seen(o.UUID) {
					nTitle := fmt.Sprintf("%s Observation", o.Taxon.PreferredCommonName)
					nBody := fmt.Sprintf("Observed On: %s\nCreated At: %s\nLocation: %s\nObscured: %s\nQuality: %s", o.ObservedOn, o.CreatedAt.Format("2006-01-02 15:04:05"), o.PlaceGuess, o.ObscuredAsString(), o.QualityGrade)
					tracker.SendNotification(nTitle, nBody, o.ID)
					fmt.Printf("Sending notification for observation ID '%d'\n", o.ID)
				}
				// Append the observation UUID to the list for this run
				thisRunUUID[o.UUID] = struct{}{}
			}
		}
		// Store this runs UUIDs in the tracker, probably not the most efficient way but does the job.
		tracker.SeenUUID = thisRunUUID
		firstRun = false
		time.Sleep(3600 * time.Second)
	}
}
