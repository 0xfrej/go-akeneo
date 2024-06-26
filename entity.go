package goakeneo

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

// ValueTypeConst
const (
	ValueTypeString = iota + 1
	ValueTypeStringCollection
	ValueTypeNumber
	ValueTypeMetric
	ValueTypePrice
	ValueTypeBoolean
	ValueTypeSimpleSelect
	ValueTypeMultiSelect
	ValueTypeTable
	ValueTypeMedia
	ValueTypeMediaSet
)

// ValueTypeName is the name of the value type
var ValueTypeName = map[int]string{
	ValueTypeString:           "string",
	ValueTypeStringCollection: "string_collection",
	ValueTypeNumber:           "number",
	ValueTypeMetric:           "metric",
	ValueTypePrice:            "price",
	ValueTypeBoolean:          "boolean",
	ValueTypeSimpleSelect:     "simple_select",
	ValueTypeMultiSelect:      "multi_select",
	ValueTypeTable:            "table",
	ValueTypeMedia:            "media_link",
	ValueTypeMediaSet:         "media_set",
}

type ErrorResponse struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Errors  []ValidationError `json:"errors,omitempty"`
}

type ValidationError struct {
	Property  string `json:"property,omitempty"`
	Message   string `json:"message,omitempty"`
	Attribute string `json:"attribute,omitempty"`
	Locale    string `json:"locale,omitempty"`
	Scope     string `json:"scope,omitempty"`
}

// Product is the struct for an akeneo product
type Product struct {
	Links                  *Links                           `json:"_links,omitempty" mapstructure:"_links"`
	UUID                   string                           `json:"uuid,omitempty" mapstructure:"uuid"` // Since Akeneo 7.0
	Identifier             string                           `json:"identifier,omitempty" mapstructure:"identifier"`
	Enabled                bool                             `json:"enabled,omitempty" mapstructure:"enabled"`
	Family                 string                           `json:"family,omitempty" mapstructure:"family"`
	Categories             []string                         `json:"categories,omitempty" mapstructure:"categories"`
	Groups                 []string                         `json:"groups,omitempty" mapstructure:"groups"`
	Parent                 string                           `json:"parent,omitempty" mapstructure:"parent"` // code of the parent product model when the product is a variant
	Values                 map[string][]ProductValue        `json:"values,omitempty" mapstructure:"values"`
	Associations           map[string]association           `json:"associations,omitempty" mapstructure:"associations"`
	QuantifiedAssociations map[string]quantifiedAssociation `json:"quantified_associations,omitempty" mapstructure:"quantified_associations"` // Since Akeneo 5.0
	Created                string                           `json:"created,omitempty" mapstructure:"created"`
	Updated                string                           `json:"updated,omitempty" mapstructure:"updated"`
	QualityScores          []QualityScore                   `json:"quality_scores,omitempty" mapstructure:"quality_scores"` // Since Akeneo 5.0,WithQualityScores must be true in the request
	Completenesses         []any                            `json:"completenesses,omitempty" mapstructure:"completenesses"` // Since Akeneo 6.0,WithCompleteness must be true in the request
	Metadata               map[string]string                `json:"metadata,omitempty" mapstructure:"metadata"`             // Enterprise Edition only
}

// Links is the struct for akeneo links
type Links struct {
	Self     Link `json:"self,omitempty"`
	First    Link `json:"first,omitempty"`
	Previous Link `json:"previous,omitempty"`
	Next     Link `json:"next,omitempty"`
	Download Link `json:"download,omitempty"`
}

// HasNext returns true if there is a next link
func (l Links) HasNext() bool {
	return l.Next.Href != ""
}

// NextOptions returns the options for the next link
func (l Links) NextOptions() url.Values {
	u, err := url.Parse(l.Next.Href)
	if err != nil {
		return nil
	}
	return u.Query()
}

// Link is the struct for an akeneo link
type Link struct {
	Href string `json:"href,omitempty"`
}

type ProductValue struct {
	Locale     *string `json:"locale" mapstructure:"locale"`
	Scope      *string `json:"scope" mapstructure:"scope"`
	Data       any     `json:"data" mapstructure:"data"`
	Links      any     `json:"_links,omitempty" mapstructure:"_links"`
	LinkedData any     `json:"linked_data,omitempty" mapstructure:"linked_data"`
}

// IsLocalized returns true if the value is localized
func (v ProductValue) IsLocalized() bool {
	return v.Locale != nil && *v.Locale != ""
}

type PimProductValue interface {
	ValueType() int
}

// ParseValue tries to parse the value to correct type
func (v ProductValue) ParseValue() (PimProductValue, error) {

	if v.Links != nil {
		if _, ok := v.Data.(*string); ok {
			result := MediaValue{
				Locale: v.Locale,
				Scope:  v.Scope,
				Data:   v.Data.(*string),
			}
			link, ok := v.Links.(map[string]interface{})
			if !ok {
				return nil, errors.New("invalid single links")
			}
			if err := mapstructure.Decode(link, &result.Links); err != nil {
				return nil, err
			}
			return result, nil
		}
		data, ok := v.Data.([]interface{})
		if !ok {
			return nil, errors.New("invalid data,should be []interface{}")
		}
		links, ok := v.Links.([]interface{})
		if !ok {
			return nil, errors.New("invalid  links slices,should be []interface{}")
		}
		s := MediaSetValue{}
		for i, d := range data {
			ds, ok := d.(string)
			if !ok {
				return nil, errors.New("invalid data elem,should be string")
			}
			s.Data = append(s.Data, ds)

			ll, ok := links[i].(map[string]interface{})
			if !ok {
				return nil, errors.New("invalid links elem,should be *Links")
			}
			link := &Links{}
			if err := mapstructure.Decode(ll, link); err != nil {
				return nil, err
			}
			s.Links = append(s.Links, link)
		}
		return s, nil
	}
	// if v.Data != nil-> simple select, multi select
	if v.LinkedData != nil {
		switch v.Data.(type) {
		case *string:
			result := SimpleSelectValue{
				Locale: v.Locale,
				Scope:  v.Scope,
				Data:   v.Data.(*string),
			}
			ld, ok := v.LinkedData.(map[string]interface{})
			if !ok {
				return nil, errors.New("invalid linked data")
			}
			if err := mapstructure.Decode(ld, &result.LinkedData); err != nil {
				return nil, err
			}
			return result, nil
		case []string:
			result := MultiSelectValue{
				Locale: v.Locale,
				Scope:  v.Scope,
				Data:   v.Data.([]string),
			}
			ld, ok := v.LinkedData.(map[string]interface{})
			if !ok {
				return nil, errors.New("invalid linked data")
			}
			if err := mapstructure.Decode(ld, &result.LinkedData); err != nil {
				return nil, err
			}
			return result, nil
		default:
			return nil, fmt.Errorf("unknown linked data type %v", v)
		}
	}
	switch v.Data.(type) {
	case *string:
		return StringValue{
			Locale: v.Locale,
			Scope:  v.Scope,
			Data:   v.Data.(*string),
		}, nil
	case []string:
		return StringCollectionValue{
			Locale: v.Locale,
			Scope:  v.Scope,
			Data:   v.Data.([]string),
		}, nil
	case *bool:
		return BooleanValue{
			Locale: v.Locale,
			Scope:  v.Scope,
			Data:   v.Data.(*bool),
		}, nil
	case *int:
		return NumberValue{
			Locale: v.Locale,
			Scope:  v.Scope,
			Data:   v.Data.(*int),
		}, nil
	case map[string]interface{}:
		d, _ := v.Data.(map[string]interface{})
		if _, ok := d["unit"]; ok {
			result := MetricValue{
				Locale: v.Locale,
				Scope:  v.Scope,
				Data: metric{
					Unit:   d["unit"].(string),
					Amount: d["amount"],
				},
			}
			return result, nil
		}
	case []interface{}:
		sd, _ := v.Data.([]interface{})
		d, ok := sd[0].(map[string]interface{})
		if !ok {
			return nil, errors.New("invalid data, should be a slice of map")
		}
		if _, ok := d["currency"]; ok {
			result := PriceValue{
				Locale: v.Locale,
				Scope:  v.Scope,
			}
			for _, item := range sd {
				pd, _ := item.(map[string]interface{})
				result.Data = append(result.Data, price{
					Currency: pd["currency"].(string),
					Amount:   pd["amount"],
				})
			}
			return result, nil
		}
		tableData := make([]map[string]any, len(sd))
		for i, item := range sd {
			d, _ := item.(map[string]interface{})
			tableData[i] = d
		}
		result := TableValue{
			Locale: v.Locale,
			Scope:  v.Scope,
			Data:   tableData,
		}
		return result, nil
	default:
	}
	return nil, errors.Errorf("unknown data type %v", v)
}

// MediaValue

type MediaValue struct {
	Locale *string `json:"locale" mapstructure:"locale"`
	Scope  *string `json:"scope" mapstructure:"scope"`
	Data   *string `json:"data" mapstructure:"data"`
	Links  *Links  `json:"_links,omitempty" mapstructure:"_links"`
}

func (MediaValue) ValueType() int {
	return ValueTypeMedia
}

// DownloadURL returns the download url of the media
func (v MediaValue) DownloadURL() string {
	if v.Links != nil {
		return v.Links.Download.Href
	}
	return ""
}

// MeidaSetValue
type MediaSetValue struct {
	Locale string   `json:"locale,omitempty" mapstructure:"locale"`
	Scope  string   `json:"scope,omitempty" mapstructure:"scope"`
	Data   []string `json:"data,omitempty" mapstructure:"data"`
	Links  []*Links `json:"_links,omitempty" mapstructure:"_links"`
}

func (MediaSetValue) ValueType() int {
	return ValueTypeMediaSet
}

// DownloadURLs returns the download urls of the media set
func (v MediaSetValue) DownloadURLs() []string {
	us := make([]string, len(v.Links))
	for i, link := range v.Links {
		us[i] = link.Download.Href
	}
	return us
}

// Hrefs returns the hrefs of the media set
func (v MediaSetValue) Hrefs() []string {
	us := make([]string, len(v.Links))
	for i, link := range v.Links {
		us[i] = link.Self.Href
	}
	return us
}

// StringValue is the struct for an akeneo text type product value
// pim_catalog_text or pim_catalog_textarea : data is a string
// pim_catalog_file or pim_catalog_image: data is the file path
// pim_catalog_date : data is a string in ISO-8601 format
type StringValue struct {
	Locale *string `json:"locale" mapstructure:"locale"`
	Scope  *string `json:"scope" mapstructure:"scope"`
	Data   *string `json:"data" mapstructure:"data"`
}

// ValueType returns the value type, see ValueTypeConst
func (StringValue) ValueType() int {
	return ValueTypeString
}

// StringCollectionValue is the struct for an akeneo collection type product value
type StringCollectionValue struct {
	Locale *string  `json:"locale" mapstructure:"locale"`
	Scope  *string  `json:"scope" mapstructure:"scope"`
	Data   []string `json:"data,omitempty" mapstructure:"data"`
}

// ValueType returns the value type, see ValueTypeConst
func (StringCollectionValue) ValueType() int {
	return ValueTypeStringCollection
}

// NumberValue is the struct for an akeneo number type product value
// pim_catalog_number : data is an int when decimal is false ,float64 string when decimal is true
// so the data will be parsed as ValueTypeString when decimal is true
type NumberValue struct {
	Locale *string `json:"locale" mapstructure:"locale"`
	Scope  *string `json:"scope" mapstructure:"scope"`
	Data   *int    `json:"data" mapstructure:"data"`
}

// ValueType returns the value type, see ValueTypeConst
func (NumberValue) ValueType() int {
	return ValueTypeNumber
}

// MetricValue is the struct for an akeneo metric type product value
// pim_catalog_metric : data amount is a float64 string when decimal is true, int when decimal is false
type MetricValue struct {
	Locale *string `json:"locale" mapstructure:"locale"`
	Scope  *string `json:"scope" mapstructure:"scope"`
	Data   metric  `json:"data" mapstructure:"data"`
}

type metric struct {
	Amount any    `json:"amount,omitempty" mapstructure:"amount"`
	Unit   string `json:"unit,omitempty" mapstructure:"unit"`
}

// ValueType returns the value type, see ValueTypeConst
func (MetricValue) ValueType() int {
	return ValueTypeMetric
}

// Amount returns the amount as string
func (v MetricValue) Amount() string {
	if f, ok := v.Data.Amount.(string); ok {
		return f
	}
	i, ok := v.Data.Amount.(int)
	if !ok {
		return ""
	}
	return strconv.Itoa(i)
}

// Unit returns the unit as string
func (v MetricValue) Unit() string {
	return v.Data.Unit
}

// PriceValue is the struct for an akeneo price type product value
// pim_catalog_price : data amount is a float64 string when decimal is true, int when decimal is false
type PriceValue struct {
	Locale *string `json:"locale" mapstructure:"locale"`
	Scope  *string `json:"scope" mapstructure:"scope"`
	Data   []price `json:"data,omitempty" mapstructure:"data"`
}

type price struct {
	Amount   any    `json:"amount,omitempty" mapstructure:"amount"`
	Currency string `json:"currency,omitempty" mapstructure:"currency"`
}

// ValueType returns the value type, see ValueTypeConst
func (PriceValue) ValueType() int {
	return ValueTypePrice
}

// Amount returns the amount as string
func (v PriceValue) Amount(currency string) string {
	for _, p := range v.Data {
		if p.Currency == currency {
			if f, ok := p.Amount.(string); ok {
				return f
			}
			i, ok := p.Amount.(int)
			if !ok {
				return ""
			}
			return strconv.Itoa(i)
		}
	}
	return ""
}

// BooleanValue is the struct for an akeneo boolean type product value
// pim_catalog_boolean : data is a bool
type BooleanValue struct {
	Locale *string `json:"locale" mapstructure:"locale"`
	Scope  *string `json:"scope" mapstructure:"scope"`
	Data   *bool   `json:"data" mapstructure:"data"`
}

// ValueType returns the value type, see ValueTypeConst
func (BooleanValue) ValueType() int {
	return ValueTypeBoolean
}

type linkedData struct {
	Attribute string            `json:"attribute,omitempty" mapstructure:"attribute"`
	Code      string            `json:"code,omitempty" mapstructure:"code"`
	Labels    map[string]string `json:"labels,omitempty" mapstructure:"labels"`
}

// SimpleSelectValue is the struct for an akeneo simple select type product value
type SimpleSelectValue struct {
	Locale     *string    `json:"locale" mapstructure:"locale"`
	Scope      *string    `json:"scope" mapstructure:"scope"`
	Data       *string    `json:"data" mapstructure:"data"`
	LinkedData linkedData `json:"linked_data,omitempty" mapstructure:"linked_data"`
}

// ValueType returns the value type, see ValueTypeConst
func (SimpleSelectValue) ValueType() int {
	return ValueTypeSimpleSelect
}

// MultiSelectValue is the struct for an akeneo multi select type product value
type MultiSelectValue struct {
	Locale     *string               `json:"locale" mapstructure:"locale"`
	Scope      *string               `json:"scope" mapstructure:"scope"`
	Data       []string              `json:"data" mapstructure:"data"`
	LinkedData map[string]linkedData `json:"linked_data,omitempty" mapstructure:"linked_data"`
}

// ValueType returns the value type, see ValueTypeConst
func (MultiSelectValue) ValueType() int {
	return ValueTypeMultiSelect
}

// TableValue is the struct for an akeneo table type product value
// pim_catalog_table : data is a []map[string]any
type TableValue struct {
	Locale *string `json:"locale" mapstructure:"locale"`
	Scope  *string `json:"scope" mapstructure:"scope"`
	Data   []map[string]any
}

// ValueType returns the value type, see ValueTypeConst
func (TableValue) ValueType() int {
	return ValueTypeTable
}

// ProductModel is the struct for an akeneo product model
type ProductModel struct {
	Links                  *Links                           `json:"_links,omitempty" mapstructure:"_links"`
	Code                   string                           `json:"code,omitempty" mapstructure:"code"`
	Family                 string                           `json:"family,omitempty" mapstructure:"family"`
	FamilyVariant          string                           `json:"family_variant,omitempty" mapstructure:"family_variant"`
	Parent                 string                           `json:"parent,omitempty" mapstructure:"parent"`
	Categories             []string                         `json:"categories,omitempty" mapstructure:"categories"`
	Values                 map[string][]ProductValue        `json:"values,omitempty" mapstructure:"values"`
	Associations           map[string]association           `json:"associations,omitempty" mapstructure:"associations"`
	QuantifiedAssociations map[string]quantifiedAssociation `json:"quantified_associations,omitempty" mapstructure:"quantified_associations"`
	Metadata               map[string]string                `json:"metadata,omitempty" mapstructure:"metadata"`
	Created                string                           `json:"created,omitempty" mapstructure:"created"`
	Updated                string                           `json:"updated,omitempty" mapstructure:"updated"`
	QulityScores           []QualityScore                   `json:"quality_scores,omitempty" mapstructure:"quality_scores"`
}

// validateBeforeCreate validates the product model before creating it
func (p ProductModel) validateBeforeCreate() error {
	if p.Code == "" {
		return errors.New("code is required")
	}
	if p.FamilyVariant == "" {
		return errors.New("family is required")
	}
	return nil
}

type association struct {
	Groups        []string `json:"groups,omitempty" mapstructure:"groups"`
	Products      []string `json:"products,omitempty" mapstructure:"products"`
	ProductModels []string `json:"product_models,omitempty" mapstructure:"product_models"`
}

// QuantifiedAssociations is the struct for an akeneo quantified associations
type quantifiedAssociation struct {
	Products      []productQuantity      `json:"products,omitempty" mapstructure:"products"`
	ProductModels []productModelQuantity `json:"product_models,omitempty" mapstructure:"product_models"`
}

type productQuantity struct {
	Identifier string `json:"identifier,omitempty" mapstructure:"identifier"`
	Quantity   int    `json:"quantity,omitempty" mapstructure:"quantity"`
}

type productModelQuantity struct {
	Code     string `json:"code,omitempty" mapstructure:"code"`
	Quantity int    `json:"quantity,omitempty" mapstructure:"quantity"`
}

// QualityScore is the struct for quality score
type QualityScore struct {
	Scope  string `json:"scope,omitempty" validate:"required"`
	Locale string `json:"locale,omitempty" validate:"required"`
	Data   string `json:"data,omitempty" validate:"required"`
}

// Family is the struct for an akeneo family
type Family struct {
	Links                 *Links              `json:"_links,omitempty" mapstructure:"_links"`
	Code                  string              `json:"code,omitempty" mapstructure:"code"`                                     // The code of the family
	Attributes            []string            `json:"attributes,omitempty" mapstructure:"attributes"`                         //  Attributes codes that compose the family
	AttributeAsLabel      string              `json:"attribute_as_label,omitempty" mapstructure:"attribute_as_label"`         // The code of the attribute used as label for the family
	AttributeAsImage      string              `json:"attribute_as_image,omitempty" mapstructure:"attribute_as_image"`         // Attribute code used as the main picture in the user interface (only since v2.fmt
	AttributeRequirements map[string][]string `json:"attribute_requirements,omitempty" mapstructure:"attribute_requirements"` //  • Attributes codes of the family that are required for the completeness calculation for the channel `channelCode`
	Labels                map[string]string   `json:"labels,omitempty" mapstructure:"labels"`                                 //  Translatable labels. Ex: {"en_US": "T-shirt", "fr_FR": "T-shirt"}
}

// FamilyVariant is the struct for an akeneo family variant
type FamilyVariant struct {
	Links                *Links                `json:"_links,omitempty" mapstructure:"_links"`
	Code                 string                `json:"code,omitempty" mapstructure:"code"`                                     // The code of the family variant
	Labels               map[string]string     `json:"labels,omitempty" mapstructure:"labels"`                                 // Translatable labels. Ex: {"en_US": "T-shirt", "fr_FR": "T-shirt"}
	VariantAttributeSets []VariantAttributeSet `json:"variant_attribute_sets,omitempty" mapstructure:"variant_attribute_sets"` // The variant attribute sets of the family variant
}

type VariantAttributeSet struct {
	Level      int      `json:"level,omitempty" mapstructure:"level"`           // The level of the variant attribute set
	Axes       []string `json:"axes,omitempty" mapstructure:"axes"`             // The axes of the variant attribute set
	Attributes []string `json:"attributes,omitempty" mapstructure:"attributes"` // The attributes of the variant attribute set
}

// Attribute is the struct for an akeneo attribute,see:
// https://api.akeneo.com/api-reference.html#Attribute
type Attribute struct {
	Links               *Links            `json:"_links,omitempty" mapstructure:"_links"`
	Code                string            `json:"code,omitempty" mapstructure:"code"`
	Type                string            `json:"type,omitempty" mapstructure:"type"`
	Labels              map[string]string `json:"labels,omitempty" mapstructure:"labels"`
	Group               string            `json:"group,omitempty" mapstructure:"group"`
	GroupLabels         map[string]string `json:"group_labels,omitempty" mapstructure:"group_labels"`
	SortOrder           *int              `json:"sort_order,omitempty" mapstructure:"sort_order"`
	Localizable         *bool             `json:"localizable,omitempty" mapstructure:"localizable"`                       // whether the attribute is localizable or not,i.e. whether it can be translated or not
	Scopable            *bool             `json:"scopable,omitempty" mapstructure:"scopable"`                             // whether the attribute is scopable or not,i.e. whether it can have different values depending on the channel or not
	AvailableLocales    []string          `json:"available_locales,omitempty" mapstructure:"available_locales"`           // the list of activated locales for the attribute values
	Unique              *bool             `json:"unique,omitempty" mapstructure:"unique"`                                 // whether the attribute value is unique or not
	UseableAsGridFilter *bool             `json:"useable_as_grid_filter,omitempty" mapstructure:"useable_as_grid_filter"` // whether the attribute can be used as a filter in the product grid or not
	MaxCharacters       *int              `json:"max_characters,omitempty" mapstructure:"max_characters"`                 // the maximum number of characters allowed for the value of the attribute
	ValidationRule      *string           `json:"validation_rule,omitempty" mapstructure:"validation_rule"`               // validation rule code to validate the attribute value
	ValidationRegexp    *string           `json:"validation_regexp,omitempty" mapstructure:"validation_regexp"`           // validation regexp to validate the attribute value
	WysiwygEnabled      *bool             `json:"wysiwyg_enabled,omitempty" mapstructure:"wysiwyg_enabled"`               // whether the attribute can have a value per channel or not
	NumberMin           *string           `json:"number_min,omitempty" mapstructure:"number_min"`                         // the minimum value allowed for the value of the attribute
	NumberMax           *string           `json:"number_max,omitempty" mapstructure:"number_max"`                         // the maximum value allowed for the value of the attribute
	DecimalsAllowed     *bool             `json:"decimals_allowed,omitempty" mapstructure:"decimals_allowed"`             // whether decimals are allowed for the attribute or not
	NegativeAllowed     *bool             `json:"negative_allowed,omitempty" mapstructure:"negative_allowed"`             // whether negative numbers are allowed for the attribute or not
	MetricFamily        *string           `json:"metric_family,omitempty" mapstructure:"metric_family"`                   // the metric family of the attribute
	DefaultMetricUnit   *string           `json:"default_metric_unit,omitempty" mapstructure:"default_metric_unit"`       // the default metric unit of the attribute
	DateMin             *string           `json:"date_min,omitempty" mapstructure:"date_min"`                             // the minimum date allowed for the value of the attribute
	DateMax             *string           `json:"date_max,omitempty" mapstructure:"date_max"`                             // the maximum date allowed for the value of the attribute
	AllowedExtensions   []string          `json:"allowed_extensions,omitempty" mapstructure:"allowed_extensions"`         // the list of allowed extensions for the value of the attribute
	MaxFileSize         *string           `json:"max_file_size,omitempty" mapstructure:"max_file_size"`                   // the maximum file size allowed for the value of the attribute
	ReferenceDataName   *string           `json:"reference_data_name,omitempty" mapstructure:"reference_data_name"`       // the reference data name of the attribute
	DefaultValue        *bool             `json:"default_value,omitempty" mapstructure:"default_value"`                   // the default value of the attribute
	TableConfiguration  []string          `json:"table_configuration,omitempty" mapstructure:"table_configuration"`       // the table configuration of the attribute
}

// AttributeOption is the struct for an akeneo attribute option,see:
type AttributeOption struct {
	Links     *Links            `json:"_links,omitempty" mapstructure:"_links"`
	Code      string            `json:"code,omitempty" mapstructure:"code"`
	Attribute string            `json:"attribute,omitempty" mapstructure:"attribute"`
	SortOrder *int              `json:"sort_order,omitempty" mapstructure:"sort_order"`
	Labels    map[string]string `json:"labels,omitempty" mapstructure:"labels"`
}

// Category is the struct for an akeneo category
type Category struct {
	Links    *Links                   `json:"_links,omitempty" mapstructure:"_links"`
	Code     string                   `json:"code,omitempty" mapstructure:"code"`
	Parent   *string                  `json:"parent,omitempty" mapstructure:"parent"`
	Updated  *string                  `json:"updated,omitempty" mapstructure:"updated"`
	Position *int                     `json:"position,omitempty" mapstructure:"position"` // since 7.0 with query parameter "with_positions=true"
	Labels   map[string]string        `json:"labels,omitempty" mapstructure:"labels"`
	Values   map[string]categoryValue `json:"values,omitempty" mapstructure:"values"`
}

// categoryValue is the struct for an akeneo category value
// todo : Data field is not yet implemented well
type categoryValue struct {
	Data          any    `json:"data,omitempty" mapstructure:"data"`           //  AttributeValue
	Type          string `json:"type,omitempty" mapstructure:"type"`           //  AttributeType
	Locale        string `json:"locale,omitempty" mapstructure:"locale"`       //  AttributeLocale
	Channel       string `json:"channel,omitempty" mapstructure:"channel"`     //  AttributeChannel
	AttributeCode string `json:"attribute,omitempty" mapstructure:"attribute"` //  AttributeCode with uuid, i.e. "description|96b88bf4-c2b7-4b64-a1f9-5d4876c02c26"
}

// Channel is the struct for an akeneo channel
type Channel struct {
	Links           *Links            `json:"_links,omitempty" mapstructure:"_links"`
	Code            string            `json:"code,omitempty" mapstructure:"code"`
	Currencies      []string          `json:"currencies,omitempty" mapstructure:"currencies"`
	Locales         []string          `json:"locales,omitempty" mapstructure:"locales"`
	CategoryTree    string            `json:"category_tree,omitempty" mapstructure:"category_tree"`
	ConversionUnits map[string]string `json:"conversion_units,omitempty" mapstructure:"conversion_units"`
	Labels          map[string]string `json:"labels,omitempty" mapstructure:"labels"`
}

// Locale is the struct for an akeneo locale
type Locale struct {
	Links   *Links `json:"_links,omitempty" mapstructure:"_links"`
	Code    string `json:"code,omitempty" mapstructure:"code"`
	Enabled bool   `json:"enabled,omitempty" mapstructure:"enabled"`
}

// MediaFile is the struct for an akeneo media file
type MediaFile struct {
	Code             string `json:"code,omitempty" mapstructure:"code"`
	OriginalFilename string `json:"original_filename,omitempty" mapstructure:"original_filename"`
	MimeType         string `json:"mime_type,omitempty" mapstructure:"mime_type"`
	Size             int    `json:"size,omitempty" mapstructure:"size"`
	Extension        string `json:"extension,omitempty" mapstructure:"extension"`
	Links            *Links `json:"_links,omitempty" mapstructure:"_links"`
}

// DownloadURL function returns the download url of the media file
func (m *MediaFile) DownloadURL() string {
	return m.Links.Download.Href
}
