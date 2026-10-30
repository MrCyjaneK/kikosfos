import QtQuick 2.6
import Sailfish.Silica 1.0

BackgroundItem {
    id: key
    property string digit
    property string letters: ""
    signal pressed()

    width: parent.width / 3
    height: Theme.itemSizeLarge
    onClicked: pressed()

    Column {
        anchors.centerIn: parent
        spacing: 0

        Label {
            anchors.horizontalCenter: parent.horizontalCenter
            text: key.digit
            font.pixelSize: Theme.fontSizeLarge
        }
        Label {
            anchors.horizontalCenter: parent.horizontalCenter
            text: key.letters
            visible: key.letters.length > 0
            color: Theme.secondaryColor
            font.pixelSize: Theme.fontSizeExtraSmall
        }
    }
}
