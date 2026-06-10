package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image/color"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	sm "github.com/flopp/go-staticmaps"
	"github.com/fogleman/gg"
	"github.com/golang/geo/s2"
	"github.com/tkrajina/gpxgo/gpx"
)

type App struct {
	Logger *slog.Logger
}

type TrackData struct {
	Path       []s2.LatLng
	FirstPoint gpx.GPXPoint
	LastPoint  gpx.GPXPoint
}

type HikeData struct {
	StartTime        time.Time
	EndTime          time.Time
	MovingTime       float64
	DistanceInMeters float64
	TimeTaken        time.Duration
	Ascent           float64
	Descent          float64
	StartLocation    Location
	EndLocation      Location
}

type Location struct {
	Locality  string
	Region    string
	Country   string
	Latitude  float64
	Longitude float64
}

func (l Location) String() string {
	return fmt.Sprintf("%s, %s", l.Locality, l.Region)
}

func (h HikeData) String() string {
	return fmt.Sprintf("from %s to %s. %.2f km, %.2fm ascent, %.2fm decent in %s", h.StartLocation, h.EndLocation, h.DistanceInMeters/1000, h.Ascent, h.Descent, h.TimeTaken)
}

func (h HikeData) MapFileName() string {
	return fmt.Sprintf("%s_map.png", h.StartTime.Format(time.RFC3339))
}

func main() {
	logger := slog.Default()
	app := App{
		Logger: logger,
	}

	inputDir := flag.String("in", "/home/jayr/Downloads/etrex", "input directory")
	flag.Parse()

	err := filepath.Walk(*inputDir,
		func(path string, info os.FileInfo, err error) error {
			ext := filepath.Ext(path)
			if ext != ".gpx" {
				return nil
			}
			app.Logger = slog.With("inputFile", path)
			hikeData := app.ProcessFile(path)
			slog.Info("=== hike", "hikeData", hikeData)
			return nil
		})
	if err != nil {
		log.Fatalf("failed walking dir %s", err.Error())
	}
}

func (app App) ProcessFile(inputFilePath string) HikeData {
	app.Logger.Info("processing file")

	inputFile, err := os.ReadFile(inputFilePath)
	if err != nil {
		app.Logger.Error("failed opening file")
	}

	// parseGpxData()
	gpxData, err := gpx.ParseBytes(inputFile)
	if err != nil {
		app.Logger.Error("failed parsing bytes")
	}
	gpxData.ReduceGpxToSingleTrack()
	gpxData.SimplifyTracks(0.3)

	trackData := ParseTrackData(gpxData)

	// hikeData.AddTrack()
	hikeData := HikeData{
		StartTime:        gpxData.TimeBounds().StartTime,
		EndTime:          gpxData.TimeBounds().EndTime,
		TimeTaken:        gpxData.TimeBounds().EndTime.Sub(gpxData.TimeBounds().StartTime),
		MovingTime:       gpxData.MovingData().MovingTime,
		DistanceInMeters: gpxData.Length2D(),
		Ascent:           gpxData.UphillDownhill().Uphill,
		Descent:          gpxData.UphillDownhill().Downhill,
	}

	// reverseGeo
	hikeData.StartLocation = ReverseGeo(trackData.FirstPoint)
	hikeData.EndLocation = ReverseGeo(trackData.LastPoint)

	// create static map
	ctx := sm.NewContext()
	ctx.SetSize(1080, 1080)

	ctx.AddObject(
		sm.NewPath(trackData.Path, color.RGBA{255, 0, 0, 255}, 4.0),
	)

	ctx.AddObject(
		sm.NewMarker(
			s2.LatLngFromDegrees(trackData.FirstPoint.GetLatitude(), trackData.FirstPoint.GetLongitude()),
			color.RGBA{0, 255, 0, 255},
			16.0,
		))

	ctx.AddObject(
		sm.NewMarker(
			s2.LatLngFromDegrees(trackData.LastPoint.GetLatitude(), trackData.LastPoint.GetLongitude()),
			color.RGBA{255, 0, 0, 255},
			16.0,
		),
	)

	img, err := ctx.Render()
	if err != nil {
		app.Logger.Error("failed to render image")
	}

	err = gg.SavePNG(hikeData.MapFileName(), img)
	if err != nil {
		app.Logger.Error("failed to save image")
	}

	return hikeData
}

func ParseTrackData(gpxFile *gpx.GPX) TrackData {
	response := TrackData{}

	for _, track := range gpxFile.Tracks {
		lastSeg := track.Segments[len(track.Segments)-1]
		response.FirstPoint = track.Segments[0].Points[0]
		response.LastPoint = lastSeg.Points[len(lastSeg.Points)-1]
		for _, seg := range track.Segments {
			for _, point := range seg.Points {
				latlng := s2.LatLngFromDegrees(point.GetLatitude(), point.GetLongitude())
				response.Path = append(response.Path, latlng)
			}
		}
	}

	return response
}

func ReverseGeo(point gpx.GPXPoint) Location {
	client := &http.Client{}
	logger := slog.Default()

	// build URL
	baseURL := "http://nominatim.openstreetmap.org/reverse"
	requestURL, err := url.Parse(baseURL)
	if err != nil {
		logger.Error("failed to parse url", "baseURL", baseURL)
	}

	q := requestURL.Query()
	q.Set("format", "geocodejson")
	q.Set("zoom", "18")
	q.Set("addressdetails", "1")
	lat := strconv.FormatFloat(point.Latitude, 'f', -1, 64)
	lon := strconv.FormatFloat(point.Longitude, 'f', -1, 64)
	q.Set("lat", lat)
	q.Set("lon", lon)
	requestURL.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", requestURL.String(), nil)
	if err != nil {
		logger.Error("failed to create request", "baseURL", baseURL, "requestURL", requestURL.String())
	}
	req.Header.Set("User-Agent", "inari")

	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to get url")
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("failed to read body")
	}

	response := ReverseGeocodeResponse{}
	json.Unmarshal(body, &response)

	locality := response.GetLocality()
	if locality == "" {
		log.Fatalf("failed to get locality :: %+v", response)
	}
	region := response.GetRegion()
	if region == "" {
		log.Fatalf("failed to get region :: %+v", response)
	}

	return Location{
		Latitude:  point.Latitude,
		Longitude: point.Longitude,
		Locality:  locality,
		Region:    region,
		Country:   response.Features[0].Properties.Geocoding.Country,
	}
}

func (res ReverseGeocodeResponse) GetRegion() string {
	fields := []string{
		res.Features[0].Properties.Geocoding.County,
		res.Features[0].Properties.Geocoding.State,
	}

	for _, loc := range fields {
		if loc != "" {
			return loc
		}
	}
	return ""
}

func (res ReverseGeocodeResponse) GetLocality() string {
	if len(res.Features) == 0 {
		return ""
	}

	feat := res.Features[0]
	fields := []string{
		feat.Properties.Geocoding.Locality,
		feat.Properties.Geocoding.Admin.Level10,
		feat.Properties.Geocoding.Admin.Level8,
		feat.Properties.Geocoding.Admin.Level5,
		feat.Properties.Geocoding.City,
	}

	for _, loc := range fields {
		if loc != "" {
			return loc
		}
	}
	return ""
}

type ReverseGeocodeResponse struct {
	Features []Feature `json:"features,omitempty"`
}

type Feature struct {
	Properties Properties `json:"properties"`
}

type Properties struct {
	Geocoding Geocoding `json:"geocoding"`
}

type Geocoding struct {
	Admin    AdminLevels `json:"admin,omitempty"`
	Locality string      `json:"locality,omitempty"`
	County   string      `json:"county,omitempty"`
	Country  string      `json:"country,omitempty"`
	State    string      `json:"state,omitempty"`
	City     string      `json:"city,omitempty"`
}

type AdminLevels struct {
	Level10 string `json:"level10,omitempty"`
	Level8  string `json:"level8,omitempty"`
	Level6  string `json:"level6,omitempty"`
	Level5  string `json:"level5,omitempty"`
	Level4  string `json:"level4,omitempty"`
}
