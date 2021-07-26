package hwevent

import (
	"io"

	"sync"

	jsoniter "github.com/json-iterator/go"

	"github.com/redhat-cne/sdk-go/pkg/types"
)

var iterPool = sync.Pool{
	New: func() interface{} {
		return jsoniter.Parse(jsoniter.ConfigFastest, nil, 1024)
	},
}

func borrowIterator(reader io.Reader) *jsoniter.Iterator {
	iter := iterPool.Get().(*jsoniter.Iterator)
	iter.Reset(reader)
	return iter
}

func returnIterator(iter *jsoniter.Iterator) {
	iter.Error = nil
	iter.Attachment = nil
	iterPool.Put(iter)
}

// ReadJSON ...
func ReadJSON(out *Event, reader io.Reader) error {
	iterator := borrowIterator(reader)
	defer returnIterator(iterator)
	return readJSONFromIterator(out, iterator)
}

// ReadDataJSON ...
func ReadDataJSON(out *Data, reader io.Reader) error {
	iterator := borrowIterator(reader)
	defer returnIterator(iterator)
	return readDataJSONFromIterator(out, iterator)
}

func readEventRecord(iter *jsoniter.Iterator) ([]EventRecord, error) {
	var result []EventRecord
	var err error
	for iter.ReadArray() {
		e := EventRecord{}
		for eField := iter.ReadObject(); eField != ""; eField = iter.ReadObject() {
			switch eField {
			case "Actions":
				e.Actions = iter.SkipAndReturnBytes()
			case "Context":
				e.Context = iter.ReadString()
			case "EventGroupId":
				e.EventGroupID = iter.ReadInt()
			case "EventId":
				e.EventID = iter.ReadString()
			case "EventTimestamp":
				e.EventTimestamp = iter.ReadString()
			case "EventType":
				e.EventType = iter.ReadString()
			case "MemberId":
				e.MemberID = iter.ReadString()
			case "Message":
				e.Message = iter.ReadString()
			case "MessageArgs":
				for iter.ReadArray() {
					arg := iter.ReadString()
					e.MessageArgs = append(e.MessageArgs, arg)
				}
			case "MessageId":
				e.MessageID = iter.ReadString()
			case "Oem":
				e.Oem = iter.SkipAndReturnBytes()
			case "OriginOfCondition":
				e.OriginOfCondition = iter.ReadString()
			case "Severity":
				e.Severity = iter.ReadString()
			case "Resolution":
				e.Resolution = iter.ReadString()
			default:
				iter.Skip()
			}
		}
		result = append(result, e)
	}

	return result, err
}

func readRedfishEvent(iter *jsoniter.Iterator) (RedfishEvent, error) {
	var result RedfishEvent
	var err error

	for key := iter.ReadObject(); key != ""; key = iter.ReadObject() {
		// Check if we have some error in our error cache
		if iter.Error != nil {
			return result, iter.Error
		}

		switch key {
		case "@odata.context":
			result.OdataContext = iter.ReadString()
		case "@odata.type":
			result.OdataType = iter.ReadString()
		case "Actions":
			result.Actions = iter.SkipAndReturnBytes()
		case "Context":
			result.Context = iter.ReadString()
		case "Description":
			result.Description = iter.ReadString()
		case "Id":
			result.ID = iter.ReadString()
		case "Name":
			result.Name = iter.ReadString()
		case "Oem":
			result.Oem = iter.SkipAndReturnBytes()
		case "Events":
			e, err := readEventRecord(iter)
			if err != nil {
				return result, err
			}
			result.Events = e
		default:
			iter.Skip()
		}
	}
	return result, err
}

// readJSONFromIterator allows you to read the bytes reader as an event
func readDataJSONFromIterator(out *Data, iter *jsoniter.Iterator) error {
	var (
		// Universally parseable fields.
		version string
		data    RedfishEvent
		// These fields require knowledge about the specversion to be parsed.
		//schemaurl jsoniter.Any
	)

	for key := iter.ReadObject(); key != ""; key = iter.ReadObject() {
		// Check if we have some error in our error cache
		if iter.Error != nil {
			return iter.Error
		}

		// If no specversion ...
		switch key {
		case "version":
			version = iter.ReadString()
		case "data":
			e, err := readRedfishEvent(iter)
			if err != nil {
				return err
			}
			data = e
		default:
			iter.Skip()
		}
	}

	if iter.Error != nil {
		return iter.Error
	}
	out.Version = version
	out.Data = &data
	return nil
}

// readJSONFromIterator allows you to read the bytes reader as an event
func readJSONFromIterator(out *Event, iterator *jsoniter.Iterator) error {
	var (
		// Universally parseable fields.
		id   string
		typ  string
		time *types.Timestamp
		data *Data
	)

	for key := iterator.ReadObject(); key != ""; key = iterator.ReadObject() {
		// Check if we have some error in our error cache
		if iterator.Error != nil {
			return iterator.Error
		}

		// If no specversion ...
		switch key {
		case "id":
			id = iterator.ReadString()
		case "type":
			typ = iterator.ReadString()
		case "time":
			time = readTimestamp(iterator)
		case "data":
			data, _ = readData(iterator)
		case "version":
		default:
			iterator.Skip()
		}
	}

	if iterator.Error != nil {
		return iterator.Error
	}
	out.Time = time
	out.ID = id
	out.Type = typ
	if data != nil {
		out.SetData(*data)
	}
	return nil
}

func readTimestamp(iter *jsoniter.Iterator) *types.Timestamp {
	t, err := types.ParseTimestamp(iter.ReadString())
	if err != nil {
		iter.Error = err
	}
	return t
}

func readData(iter *jsoniter.Iterator) (*Data, error) {
	data := &Data{
		Version: "",
		Data:    nil,
	}

	for key := iter.ReadObject(); key != ""; key = iter.ReadObject() {
		// Check if we have some error in our error cache
		if iter.Error != nil {
			return data, iter.Error
		}
		switch key {
		case "version":
			data.Version = iter.ReadString()
		case "data":
			e, err := readRedfishEvent(iter)
			if err != nil {
				return data, err
			}
			data.Data = &e
		default:
			iter.Skip()
		}
	}

	return data, nil
}

// UnmarshalJSON implements the json unmarshal method used when this type is
// unmarshaled using json.Unmarshal.
func (e *Event) UnmarshalJSON(b []byte) error {
	iterator := jsoniter.ConfigFastest.BorrowIterator(b)
	defer jsoniter.ConfigFastest.ReturnIterator(iterator)
	return readJSONFromIterator(e, iterator)
}

// UnmarshalJSON implements the json unmarshal method used when this type is
// unmarshaled using json.Unmarshal.
func (d *Data) UnmarshalJSON(b []byte) error {
	iterator := jsoniter.ConfigFastest.BorrowIterator(b)
	defer jsoniter.ConfigFastest.ReturnIterator(iterator)
	return readDataJSONFromIterator(d, iterator)
}
