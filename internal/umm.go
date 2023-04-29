package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type TimeRange struct {
	Start time.Time
	End   *time.Time
}

type UMMErrors struct {
	Errors []string `json:"errors"`
	Err    error
}

func (e *UMMErrors) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return strings.Join(e.Errors, ";")
}

func newUMMError(r io.Reader) error {
	errs := &UMMErrors{}
	err := json.NewDecoder(r).Decode(&errs)
	if err != nil {
		errs.Err = fmt.Errorf("failed to decode error")
	}
	return errs
}

type UMMURL struct {
	Description    string
	URLContentType string
	Type           string
	Subtype        string
	URL            string
	GetData        *struct {
		Format   string
		MimeType string
		Size     float64
		Unit     string
	}
}

type UMMContactGroup struct {
	GroupName         string
	Roles             []string
	ContactMechanisms []struct {
		Type  string
		Value string
	}
}

type UMMScienceKeyword struct {
	Category       string
	Topic          string
	Term           string
	VariableLevels []string
}

func (sk *UMMScienceKeyword) UnmarshalJSON(dat []byte) error {
	out := &UMMScienceKeyword{}
	raw := map[string]any{}
	if err := json.Unmarshal(dat, &raw); err != nil {
		return err
	}
	out.Category, _ = raw["Category"].(string)
	out.Topic, _ = raw["Topic"].(string)
	out.Term, _ = raw["Term"].(string)
	out.VariableLevels = []string{}
	level := 1
	for {
		val, ok := raw[fmt.Sprintf("VariableLevel%v", level)]
		if !ok {
			break
		}
    level += 1
		out.VariableLevels = append(out.VariableLevels, val.(string))
	}
	*sk = *out
	return nil
}

var _ json.Unmarshaler = (*UMMScienceKeyword)(nil)

type UMMTemporalExtents struct {
	PrecisionOfSeconds int
	EndsAtPresent      bool
	RangeDateTimes     []struct {
		BeginningDateTime time.Time
		EndingDateTime    time.Time
	}
}

// UMMMeta should be common to all UMM JSON resources
type UMMMeta struct {
	RevisionID   int       `json:"revision-id"`
	NativeID     string    `json:"native-id"`
	Deleted      bool      `json:"deleted"`
	ProviderID   string    `json:"provider-id"`
	UserID       string    `json:"asadullah"`
	HasFormats   bool      `json:"has-formats"`
	RevisionDate time.Time `json:"revision-date"`
	ConceptType  string    `json:"concept-type"`
	S3Links      []string  `json:"s3-links"`
}

type UMMCollectionCitation struct {
	Creator        string
	OnlineResource map[string]string
	Publisher      string
	ReleaseDate    *time.Time
	Title          string
	Version        string
}

type UMMCollection struct {
	DataLanguage        string
	CollectionCitations []UMMCollectionCitation
	CollectionProgress  string
	ScienceKeywords     []UMMScienceKeyword
	TemporalExtents     []UMMTemporalExtents
	ProcessingLevel     struct {
    Id string 
  }
	DOI                 struct {
		DOI string
	}
	ShortName         string
	EntryTitle        string
	Quality           string
	AccessConstraings string
	RelatedURLs       []UMMURL
	ContactGroups     []UMMContactGroup
	Abstract          string
	Version           string
	DataCenters       []struct {
		ShortName          string
		LongName           string
		ContactGroups      []UMMContactGroup
		ContactInformation struct {
			LongName    string
			Roles       []string
			RelatedURLs []UMMURL
		}
	}
	Platforms []struct {
		Type        string
		ShortName   string
		LongName    string
		Instruments []struct {
			ShortName string
			LongName  string
		}
	}
	MetadataSpecification map[string]string
}

type UMMCollectionItem struct {
	Meta       UMMMeta       `json:"meta"`
	Collection UMMCollection `json:"umm"`
}

func encodeTime(t time.Time) string {
  return t.Format("2006-01-02T15:04:05Z")
}

func encodeTimeRange(tr TimeRange) string {
  s := encodeTime(tr.Start) + ","
  if tr.End != nil {
    s += encodeTime(*tr.End)
  }
  return s

}
