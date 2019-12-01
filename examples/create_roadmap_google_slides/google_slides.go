// Go example that covers:
// Quickstart: https://developers.google.com/slides/quickstart/go
// Basic writing: adding a text box to slide: https://developers.google.com/slides/samples/writing
// Using SDK: https://github.com/google/google-api-go-client/blob/master/slides/v1/slides-gen.go
// Creating and Managing Presentations https://developers.google.com/slides/how-tos/presentations
// Adding Shapes and Text to a Slide: https://developers.google.com/slides/how-tos/add-shape#example
package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/grokify/gotilla/config"
	"github.com/grokify/gotilla/fmt/fmtutil"
	tu "github.com/grokify/gotilla/time/timeutil"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/api/slides/v1"

	om "github.com/grokify/oauth2more"
	omg "github.com/grokify/oauth2more/google"

	"github.com/grokify/go-aha/ahautil"
	aha "github.com/grokify/go-aha/client"
	"github.com/grokify/gocharts/data/roadmap"
	"github.com/grokify/googleutil/slidesutil/v1"
)

var (
	GoogleSlideUnitPoint = "PT"
)

type Options struct {
	EnvFile              string `short:"e" long:"env" description:"Env filepath"`
	Products             string `short:"p" long:"productSlugs" description:"Aha Product Slugs" required:"true"`
	ReleaseQuarterBegin  int32  `short:"b" long:"begin" description:"Begin Quarter" required:"true"`
	ReleaseQuarterFinish int32  `short:"f" long:"finish" description:"Finish Quarter" required:"true"`
	NewTokenRaw          []bool `short:"n" long:"newtoken" description:"Retrieve new token"`
}

func (opt *Options) NewToken() bool {
	if len(opt.NewTokenRaw) > 0 {
		return true
	}
	return false
}

/*
func NewClientFileStore(
	credentials []byte,
	scopes []string,
	tokenPath string,
	useDefaultDir, forceNewToken bool) (*http.Client, error) {
	conf, err := oug.ConfigFromBytes(credentials, scopes)
	if err != nil {
		return nil, err
	}
	tokenStore, err := ou.NewTokenStoreFileDefault(tokenPath, useDefaultDir, 0700)
	if err != nil {
		return nil, err
	}
	return ou.NewClientWebTokenStore(context.Background(), conf, tokenStore, forceNewToken)
}
*/

func NewGoogleClient(forceNewToken bool) (*http.Client, error) {
	if 1 == 1 {
		return omg.NewClientFileStore(
			[]byte(omg.ClientSecretEnv),
			[]string{omg.ScopeDrive, omg.ScopePresentations},
			"slides.googleapis.com-go-quickstart.json",
			true,
			forceNewToken,
		)
	}

	conf, err := omg.ConfigFromEnv(omg.ClientSecretEnv,
		[]string{slides.DriveScope, slides.PresentationsScope})
	if err != nil {
		return nil, err
	}

	tokenFile := "slides.googleapis.com-go-quickstart.json"
	tokenStore, err := om.NewTokenStoreFileDefault(tokenFile, true, 0700)
	if err != nil {
		return nil, err
	}

	return om.NewClientWebTokenStore(context.Background(), conf, tokenStore, forceNewToken)
}

func FilterFeatures(features []aha.Feature, tagFilters []string) []aha.Feature {
	tagFiltersMap := map[string]int{}
	for _, tag := range tagFilters {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if len(tag) > 0 {
			tagFiltersMap[tag] = 1
		}
	}
	filteredFeatures := []aha.Feature{}
FEATURE:
	for _, feat := range features {
		for _, tagTry := range feat.Tags {
			tagTry = strings.ToLower(strings.TrimSpace(tagTry))
			if len(tagTry) == 0 {
				continue
			}
			for tagFilter := range tagFiltersMap {
				if tagFilter == tagTry {
					filteredFeatures = append(filteredFeatures, feat)
					continue FEATURE
				}
			}
		}
	}
	return filteredFeatures
}

func TagFilterFeatureMap(featuresMap map[string]*aha.Feature, tagFilters []string) map[string]map[string]aha.Feature {
	tagFiltersMap := map[string]int{}
	for _, tag := range tagFilters {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if len(tag) > 0 {
			tagFiltersMap[tag] = 1
		}
	}

	tagFilterFeatureMap := map[string]map[string]aha.Feature{}
FEATURES:
	for id, feat := range featuresMap {
		for _, tagTry := range feat.Tags {
			tagTry = strings.ToLower(strings.TrimSpace(tagTry))
			if _, ok := tagFiltersMap[tagTry]; ok {
				if _, ok2 := tagFilterFeatureMap[tagTry]; !ok2 {
					tagFilterFeatureMap[tagTry] = map[string]aha.Feature{}
				}
				tagFilterFeatureMap[tagTry][id] = *feat
				continue FEATURES
			}
		}
	}
	return tagFilterFeatureMap
}

func NewTagsFeatures(tagIdFeatureMap map[string]map[string]aha.Feature, filterTags []string) []TagFeatures {
	tagFeaturesSlice := []TagFeatures{}
	//featureMapsArr := []map[string]aha.Feature{}
	for _, filterTag := range filterTags {
		filterTag = strings.ToLower(strings.TrimSpace(filterTag))
		if fMap, ok := tagIdFeatureMap[filterTag]; ok {
			tagFeaturesSlice = append(
				tagFeaturesSlice,
				TagFeatures{
					Tag:      filterTag,
					Features: fMap})
			//featureMapsArr = append(featureMapsArr, fMap)
		}
	}
	return tagFeaturesSlice
}

type TagsFeaturesSet struct {
	TagIdFeatureMap map[string]map[string]aha.Feature
	TagsFeatures    []TagFeatures
}

func NewTagsFeaturesSet(featuresMap map[string]*aha.Feature, tagFilters []string) TagsFeaturesSet {
	tfs := TagsFeaturesSet{}
	tfs.TagIdFeatureMap = TagFilterFeatureMap(featuresMap, tagFilters)
	tfs.TagsFeatures = NewTagsFeatures(tfs.TagIdFeatureMap, tagFilters)
	return tfs
}

type TagFeatures struct {
	Tag      string
	Features map[string]aha.Feature // map[feature.id]feature
}

func NewRoadmapCanvasAha(featuresSet ahautil.FeatureSet, yyyyq1, yyyyq2 int32) (*roadmap.Canvas, error) {
	yyyyq1, yyyyq2 = tu.MinMaxInt32([]int32{yyyyq1, yyyyq2})
	log.Info()
	//itemsRM := []roadmap.Item{}
	can := roadmap.Canvas{}

	err := can.SetMinMaxQuarter(yyyyq1, yyyyq2)
	if err != nil {
		return nil, err
	}
	can.SetRange(200)

	if 1 == 0 { // for debug only
		rng, err := can.Range.CellRange()
		if err != nil {
			return nil, err
		}
		fmtutil.PrintJSON(rng)
	}
	log.Info("IN_NewRoadmapCanvasAha_START_FeatureMap")
	for _, feat := range featuresSet.FeatureMap {
		// fmtutil.PrintJSON(feat)
		minTime, err := tu.FirstNonZeroTimeParsed(tu.RFC3339FullDate, []string{feat.StartDate, feat.Release.StartDate})
		if err != nil {
			return nil, fmt.Errorf("Feature+Release has no Feature.StartDate or Feature.Release.StartDate")
		}
		maxTime, err := tu.FirstNonZeroTimeParsed(tu.RFC3339FullDate, []string{feat.DueDate, feat.Release.ReleaseDate})
		if err != nil {
			return nil, fmt.Errorf("Feature+Release has no Feature.DueDate or Feature.Release.ReleaseDate")
		}
		// fmt.Printf("MIN MAX [%v][%v]", minTime, maxTime)

		item := roadmap.Item{
			MinTime: minTime,
			MaxTime: maxTime,
			Name:    feat.Name,
		}
		//itemsRM = append(itemsRM, item)
		can.AddItem(item)
	}
	//panic("Z")
	log.Info("IN_NewRoadmapCanvasAha_END_FeatureMap")

	//fmtutil.PrintJSON(itemsRM)
	log.Info("IN_NewRoadmapCanvasAha_BEG_InflateItems")
	err = can.InflateItems()
	if err != nil {
		return nil, err
	}
	log.Info("IN_NewRoadmapCanvasAha_END_InflateItems")
	log.Info("IN_NewRoadmapCanvasAha_BEG_BuildRows")
	can.BuildRows()
	log.Info("IN_NewRoadmapCanvasAha_END_BuildRows")
	return &can, nil
}

func main() {
	opts := Options{}
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Fatal(err)
	}

	err = config.LoadDotEnvSkipEmpty(opts.EnvFile, os.Getenv("ENV_PATH"))
	if err != nil {
		log.Fatal(err)
	}

	featuresPath := "../get_features_by_release_and_date/_features.json"

	//featuersMap := map[string]aha.Feature{}
	filterArr := []string{"rmglip", "rmcc", "rmcpaas", "rmeco", "rmreq"}
	filterArr = []string{"Engage Voice", "Engage Digital"}
	//filterMap := map[string]int{"rmglip": 1, "rmcc": 1, "rmcpaas": 1, "rmeco": 1, "rmreq": 1}
	swimlaneTags := filterArr

	log.Info("START_ReadFeatureSet")
	featuresSet, err := ahautil.ReadFeatureSet(featuresPath)
	if err != nil {
		log.Fatal(err)
	}

	fmtutil.PrintJSON(featuresSet.FeatureMap)

	log.Info("START_NewTagsFeaturesSet")
	tagsFeaturesSet := NewTagsFeaturesSet(featuresSet.FeatureMap, swimlaneTags)
	featuresMap2 := tagsFeaturesSet.TagIdFeatureMap

	fmtutil.PrintJSON(featuresMap2)

	log.Infof("START_NewRoadmapCanvasAha BEG[%v] END[%v]", opts.ReleaseQuarterBegin, opts.ReleaseQuarterFinish)
	can, err := NewRoadmapCanvasAha(*featuresSet, opts.ReleaseQuarterBegin, opts.ReleaseQuarterFinish)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("FINISH_NewRoadmapCanvasAha")

	fmtutil.PrintJSON(can)
	fmt.Println(len(can.Rows))
	//panic("Z")

	gcfsConfig := omg.GoogleConfigFileStore{
		Scopes:        []string{omg.ScopeDrive, omg.ScopePresentations},
		TokenPath:     "google_aha-roadmap_token.json",
		UseDefaultDir: true,
		ForceNewToken: opts.NewToken()}

	err = gcfsConfig.LoadCredentialsBytes([]byte(os.Getenv(omg.EnvGoogleAppCredentials)))
	if err != nil {
		log.Fatal(err)
	}

	//googClient, err := NewGoogleClient(forceNewToken)
	googClient, err := gcfsConfig.Client()
	if err != nil {
		log.Fatal("Unable to get Google Client")
	}

	gsc, err := slidesutil.NewGoogleSlidesService(googClient)
	if err != nil {
		log.Fatal(err)
	}

	t := time.Now().UTC()
	slideName := fmt.Sprintf("GOLANG TEST PRES %v\n", t.Format(time.RFC3339))
	fmt.Printf("Slide Name: %v", slideName)

	res, err := gsc.PresentationsService.Create(
		&slides.Presentation{Title: slideName},
	).Do()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("CREATED Presentation with Id %v\n", res.PresentationId)

	for i, slide := range res.Slides {
		fmt.Printf("- Slide #%d id %v contains %d elements.\n", (i + 1),
			slide.ObjectId,
			len(slide.PageElements))
	}

	pageId := res.Slides[0].ObjectId

	//requests := []*slides.Request{}
	//items := []string{"Item #1", "Item #2"}
	//locX := 150.0
	//locY := 50.0
	//boxWidth := 550.0
	//boxHeight := 25.0

	log.Info("START_GoogleSlideDrawRoadmap")
	requestsRoadmap, err := slidesutil.GoogleSlideDrawRoadmap(
		pageId, *can, slidesutil.DefaultSlideCanvasInfo())
	if err != nil {
		log.Fatal(err)
	}

	resu, err := gsc.PresentationsService.BatchUpdate(
		res.PresentationId,
		&slides.BatchUpdatePresentationRequest{
			Requests: requestsRoadmap,
		}).Do()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resu.PresentationId)

	fmt.Println("DONE")
}

/*

	locYHeight := boxHeight + 5.0
	for i, itemText := range items {
		elementId := fmt.Sprintf("item%v", i)
		locYThis := locY + locYHeight*float64(i)
		requests = append(requests, su.TextBoxRequestsSimple(
			pageId, elementId, itemText, fgColor, bgColor,
			boxWidth, boxHeight, locX, locYThis)...)
	}
*/
