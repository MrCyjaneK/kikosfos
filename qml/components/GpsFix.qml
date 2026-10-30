import QtQuick 2.6
import Sailfish.Silica 1.0

Column {
    id: root
    property string lat
    property string lon
    property string accuracy
    property string message

    width: parent.width
    spacing: Theme.paddingSmall
    visible: root.lat !== "" || root.message !== ""
    height: visible ? implicitHeight : 0

    Row {
        x: Theme.horizontalPageMargin
        spacing: Theme.paddingLarge
        visible: root.lat !== ""

        Label {
            text: root.lat
            font.pixelSize: Theme.fontSizeSmall
            color: Theme.primaryColor
        }

        Label {
            text: root.lon
            font.pixelSize: Theme.fontSizeSmall
            color: Theme.secondaryColor
        }

        Row {
            spacing: Theme.paddingSmall
            visible: root.accuracy !== ""

            Label {
                id: acc
                text: root.accuracy
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.secondaryColor
            }

            Label {
                anchors.baseline: acc.baseline
                text: "m"
                font.pixelSize: Theme.fontSizeExtraSmall
                color: Theme.secondaryColor
            }
        }
    }

    Label {
        x: Theme.horizontalPageMargin
        width: parent.width - 2 * x
        visible: root.message !== ""
        wrapMode: Text.Wrap
        text: root.message
        color: Theme.highlightColor
        font.pixelSize: Theme.fontSizeSmall
    }
}
