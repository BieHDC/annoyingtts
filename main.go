package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
)

var (
	textlenlimit int = 200 // they might change this in the future
	verbose      bool
)

func main() {
	text := flag.String("in", "", "pass text in instead of reading from stdin")
	voice := flag.String("voice", "", "the voice to use (blank means random)")
	printvoices := flag.Bool("voices", false, "print out the available voices and exit")
	flag.BoolVar(&verbose, "verbose", false, fmt.Sprintf("helps you figure out the chunking when stuff sounds weird and cut off (currently %d chars)", textlenlimit))
	flag.Parse()
	if *printvoices == true {
		fmt.Fprintf(os.Stderr, "Voices:\n\t%s\n", strings.Join(voices, "\n\t"))
		return
	}

	if *voice == "" {
		*voice = voices[rand.Intn(len(voices))]
		fmt.Fprintf(os.Stderr, "Selected Random Voice: %s\n", *voice)
	}

	if *text == "" {
		fmt.Fprintf(os.Stderr, "Reading from stdin\n")
		t, _ := io.ReadAll(os.Stdin)
		*text = string(t)
		fmt.Fprintf(os.Stderr, "Done reading from stdin\n")
	}

	ret, err := tts(*text, *voice)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate voice: %s\n", err)
		return
	}

	io.Copy(os.Stdout, ret)
	fmt.Fprintf(os.Stderr, "Finished\n")
}

// define the endpoint data with URLs and corresponding response keys
type endpoint struct {
	url          string
	response     string
	successfield string
}

var endpoints = []endpoint{
	{
		url:          "https://tiktok-tts.weilnet.workers.dev/api/generation",
		response:     "data",
		successfield: "success",
	},
	{
		url:      "https://gesserit.co/api/tiktok-tts",
		response: "base64",
	},
}

// define available voices for text-to-speech conversion
var voices = []string{
	// DISNEY VOICES
	"en_us_ghostface",    // Ghost Face
	"en_us_chewbacca",    // Chewbacca
	"en_us_c3po",         // C3PO
	"en_us_stitch",       // Stitch
	"en_us_stormtrooper", // Stormtrooper
	"en_us_rocket",       // Rocket
	// ENGLISH VOICES
	"en_au_001", // English AU - Female
	"en_au_002", // English AU - Male
	"en_uk_001", // English UK - Male 1
	"en_uk_003", // English UK - Male 2
	"en_us_001", // English US - Female (Int. 1)
	"en_us_002", // English US - Female (Int. 2)
	"en_us_006", // English US - Male 1
	"en_us_007", // English US - Male 2
	"en_us_009", // English US - Male 3
	"en_us_010", // English US - Male 4
	// EUROPE VOICES
	"fr_001", // French - Male 1
	"fr_002", // French - Male 2
	"de_001", // German - Female
	"de_002", // German - Male
	"es_002", // Spanish - Male
	// AMERICA VOICES
	"es_mx_002", // Spanish MX - Male
	"br_001",    // Portuguese BR - Female 1
	"br_003",    // Portuguese BR - Female 2
	"br_004",    // Portuguese BR - Female 3
	"br_005",    // Portuguese BR - Male
	// ASIA VOICES
	"id_001", // Indonesian - Female
	"jp_001", // Japanese - Female 1
	"jp_003", // Japanese - Female 2
	"jp_005", // Japanese - Female 3
	"jp_006", // Japanese - Male
	"kr_002", // Korean - Male 1
	"kr_003", // Korean - Female
	"kr_004", // Korean - Male 2
	// SINGING VOICES
	"en_female_f08_salut_damour", // Alto
	"en_male_m03_lobby",          // Tenor
	"en_female_f08_warmy_breeze", // Warmy Breeze
	"en_male_m03_sunshine_soon",  // Sunshine Soon
	// OTHER
	"en_male_narration",   // narrator
	"en_male_funny",       // wacky
	"en_female_emotional", // peaceful
	// More Singing
	"en_male_sing_deep_jingle",
	"en_male_sing_funny_it_goes_up",
	"en_male_m2_xhxs_m03_silly",
	"en_male_sing_funny_thanksgiving",
	"en_female_ht_f08_glorious",
	"en_female_ht_f08_wonderful_world",
	"en_female_ht_f08_halloween",
	"en_female_ht_f08_newyear",
	"en_female_f08_twinkle",
	// More others unsorted
	"en_male_jomboy",
	"en_female_samc",
	"en_female_makeup",
	"en_male_cody",
	"en_male_grinch",
	"en_female_richgirl",
	"en_male_ashmagic",
	"en_male_jarvis",
	"en_male_ukneighbor",
	"en_male_olantekkers",
	"en_female_shenna",
	"en_male_ukbutler",
	"en_male_trevor",
	"en_female_pansino",
	"en_male_m03_classical",
	"en_male_cupid",
	"en_female_betty",
	"en_male_m2_xhxs_m03_christmas",
	"en_female_grandma",
	"en_male_santa_narration",
	"en_male_santa_effect",
	"en_male_wizard",
	"en_male_ghosthost",
	"en_female_madam_leota",
	"bp_female_ivete",
	"bp_female_ludmilla",
	"pt_female_lhays",
	"pt_female_laizza",
	"pt_male_bueno",
	"jp_female_fujicochan",
	"jp_female_hasegawariona",
	"jp_male_keiichinakano",
	"jp_female_oomaeaika",
	"jp_male_yujinchigusa",
	"jp_female_shirou",
	"jp_male_tamawakazuki",
	"jp_female_kaorishoji",
	"jp_female_yagishaki",
	"jp_male_hikakin",
	"jp_female_rei",
	"jp_male_shuichiro",
	"jp_male_matsudake",
	"jp_female_machikoriiita",
	"jp_male_matsuo",
	"jp_male_osada",
	"BV074_streaming",
	"BV075_streaming",
}

// define the text-to-speech function
func tts(text, voice string) (io.Reader, error) {
	type jsonreq struct {
		Text  string `json:"text"`
		Voice string `json:"voice"`
	}

	// specified voice is valid
	if !slices.Contains(voices, voice) {
		return nil, fmt.Errorf("unknown voice: %s", voice)
	}

	if text == "" {
		return nil, errors.New("empty text")
	}

	// split the text into chunks
	chunks := splitText(text)
	if verbose {
		for i, c := range chunks {
			fmt.Fprintf(os.Stderr, "Chunk %d len(%d): %s\n", i, len(c), c)
		}
	}

	client := &http.Client{}
	for _, ep := range endpoints {
		endpointValid := true

		// empty list to store the data from the reqeusts
		var audioData []string

		// generate audio for each chunk
		for _, chunk := range chunks {
			if !endpointValid {
				break
			}

			rr := jsonreq{
				Text:  chunk,
				Voice: voice,
			}
			var buf bytes.Buffer
			err := json.NewEncoder(&buf).Encode(rr)
			if err != nil {
				return nil, fmt.Errorf("failed to encode text \"%s\": %w", chunk, err)
			}
			req, err := http.NewRequest(http.MethodPost, ep.url, &buf)
			req.Header.Add("Content-Type", "application/json")
			if err != nil {
				return nil, fmt.Errorf("failed to make request for text \"%s\": %w", chunk, err)
			}

			resp, err := client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("failed to do request for text \"%s\": %w", chunk, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				//x, _ := io.ReadAll(resp.Body)
				//fmt.Fprintf(os.Stderr, "%v\n", string(x))
				//fmt.Fprintf(os.Stderr, "bad endpoint: %s\n", ep.url)
				endpointValid = false
				continue
			}

			var epresp = make(map[string]any)
			err = json.NewDecoder(resp.Body).Decode(&epresp)
			if err != nil {
				//x, _ := io.ReadAll(resp.Body)
				//fmt.Fprintf(os.Stderr, "%v\n", x)
				return nil, fmt.Errorf("failed to unmarshal response to \"%s\": %w", chunk, err)
			}

			if ep.successfield != "" {
				b, ok := epresp[ep.successfield].(bool)
				if !ok || !b {
					//fmt.Fprintf(os.Stderr, "bad endpoint: %s\n", ep.url)
					endpointValid = false
					continue
				}
			}

			ret, ok := epresp[ep.response].(string)
			if !ok {
				return nil, fmt.Errorf("failed to access response data for \"%s\": field empty! got: %v", chunk, epresp)
			}
			audioData = append(audioData, ret)
		}

		if !endpointValid {
			continue
		}

		// concatenate audio data from all chunks and decode from base64
		bb, err := base64.StdEncoding.DecodeString(strings.Join(audioData, ""))
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64: %w", err)
		}

		// break after processing a valid endpoint
		return bytes.NewReader(bb), nil

	}

	return nil, errors.New("no endpoint provided the requested service")
}

// i didnt make this and i wont bother to question it.
// define a function to split the text into chunks of maximum 300 characters or less
var splitterexp = regexp.MustCompile(".*?[.!?:;]|.+")
var splitterexp2 = regexp.MustCompile(".*?[ ]|.+")

func splitText(text string) []string {
	// split the text into chunks based on punctuation marks
	// change the regex [.,!?:;-] to add more seperation points
	var sc []string
	for _, chunk := range splitterexp.FindAllString(text, -1) {
		// iterate through the chunks to check for their lengths
		if len(chunk) > textlenlimit {
			// Split chunk further into smaller parts
			sc = append(sc, splitterexp2.FindAllString(chunk, -1)...)
		} else {
			sc = append(sc, chunk)
		}
	}
	//for i, c := range sc {
	//	fmt.Fprintf(os.Stderr, "Subchunk %d len(%d): %s\n", i, len(c), c)
	//}

	var mergedchunks []string
	currentchunk := ""
	for _, sepchunk := range sc {
		// check if adding the current chunk would exceed the limit of 300 characters
		if len(currentchunk)+len(sepchunk) <= textlenlimit {
			currentchunk += sepchunk
		} else {
			// start a new merged chunk
			mergedchunks = append(mergedchunks, currentchunk)
			currentchunk = sepchunk
		}
	}

	mergedchunks = append(mergedchunks, currentchunk)
	return mergedchunks
}
