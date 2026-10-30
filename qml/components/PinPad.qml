import QtQuick 2.6
import Sailfish.Silica 1.0

Item {
    id: pad
    signal digit(string text)
    signal backspace()
    signal accept()

    width: parent.width
    height: keys.height
    opacity: enabled ? 1.0 : 0.4

    Column {
        id: keys
        width: parent.width

        Row {
            width: parent.width
            PinKey { digit: "1"; onPressed: pad.digit(digit) }
            PinKey { digit: "2"; letters: "ABC"; onPressed: pad.digit(digit) }
            PinKey { digit: "3"; letters: "DEF"; onPressed: pad.digit(digit) }
        }
        Row {
            width: parent.width
            PinKey { digit: "4"; letters: "GHI"; onPressed: pad.digit(digit) }
            PinKey { digit: "5"; letters: "JKL"; onPressed: pad.digit(digit) }
            PinKey { digit: "6"; letters: "MNO"; onPressed: pad.digit(digit) }
        }
        Row {
            width: parent.width
            PinKey { digit: "7"; letters: "PQRS"; onPressed: pad.digit(digit) }
            PinKey { digit: "8"; letters: "TUV"; onPressed: pad.digit(digit) }
            PinKey { digit: "9"; letters: "WXYZ"; onPressed: pad.digit(digit) }
        }
        Row {
            width: parent.width
            PinKey { digit: "Del"; onPressed: pad.backspace() }
            PinKey { digit: "0"; onPressed: pad.digit(digit) }
            PinKey { digit: "OK"; onPressed: pad.accept() }
        }
    }
}
