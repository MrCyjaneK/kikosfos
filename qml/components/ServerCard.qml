import QtQuick 2.6
import Sailfish.Silica 1.0

Rectangle {
    id: root
    property string host
    property string status
    property bool compatible
    property bool hasPrice
    property string baseAmount
    property string baseEuro
    property string perKmAmount
    property string perKmEuro
    property bool highlighted

    radius: Theme.paddingSmall
    color: root.highlighted
           ? Theme.highlightBackgroundColor
           : Theme.rgba(Theme.primaryColor, 0.06)
    implicitHeight: body.height + 2 * Theme.paddingMedium
    height: implicitHeight

    Column {
        id: body
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.margins: Theme.paddingMedium
        spacing: Theme.paddingMedium

        Item {
            width: parent.width
            height: Math.max(hostLabel.implicitHeight, statusLabel.implicitHeight)

            Label {
                id: hostLabel
                anchors.left: parent.left
                anchors.right: statusLabel.left
                anchors.rightMargin: Theme.paddingLarge
                anchors.verticalCenter: parent.verticalCenter
                text: root.host
                font.pixelSize: Theme.fontSizeMedium
                color: root.highlighted ? Theme.highlightColor : Theme.primaryColor
                truncationMode: TruncationMode.Fade
                verticalAlignment: Text.AlignVCenter
            }

            Label {
                id: statusLabel
                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                text: root.status
                font.pixelSize: Theme.fontSizeSmall
                color: root.compatible
                       ? (root.highlighted ? Theme.secondaryHighlightColor : Theme.secondaryColor)
                       : Theme.highlightColor
            }
        }

        Label {
            id: captionMeasure
            visible: false
            text: "Per km"
            font.pixelSize: Theme.fontSizeSmall
        }

        Label {
            visible: !root.hasPrice
            width: parent.width
            text: "No price"
            font.pixelSize: Theme.fontSizeSmall
            color: root.highlighted ? Theme.secondaryHighlightColor : Theme.secondaryColor
        }

        Column {
            width: parent.width
            spacing: Theme.paddingSmall
            visible: root.hasPrice

            PriceLine {
                caption: "Base"
                captionWidth: captionMeasure.implicitWidth
                amount: root.baseAmount
                euro: root.baseEuro
                highlighted: root.highlighted
            }

            PriceLine {
                caption: "Per km"
                captionWidth: captionMeasure.implicitWidth
                amount: root.perKmAmount
                euro: root.perKmEuro
                highlighted: root.highlighted
            }
        }
    }
}
