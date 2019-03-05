package plugin

import (
	"testing"

	. "github.com/stretchr/testify/assert"
)

func TestStringEncodeDecode(t *testing.T) {
	values := []interface{}{
		nil,
		"",
		"hello",
		"special chars >☺😂<",
	}

	codec := StringCodec{}

	for _, v := range values {
		data, err := codec.EncodeMessage(v)
		if err != nil {
			t.Fatal(err)
		}
		v2, err := codec.DecodeMessage(data)
		if err != nil {
			t.Fatal(err)
		}
		Equal(t, v, v2)
	}
}

//   group('String codec', () {
//     test('ByteData with offset', () {
//       const MessageCodec<String> string = StringCodec();
//       final ByteData helloWorldByteData = string.encodeMessage('hello world');
//       final ByteData helloByteData = string.encodeMessage('hello');
//
//       final ByteData offsetByteData = ByteData.view(
//           helloWorldByteData.buffer,
//           helloByteData.lengthInBytes,
//           helloWorldByteData.lengthInBytes - helloByteData.lengthInBytes,
//       );
//
//       expect(string.decodeMessage(offsetByteData), ' world');
//     });
// });
