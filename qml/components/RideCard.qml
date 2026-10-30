import QtQuick 2.6
import Sailfish.Silica 1.0

Rectangle {
    id: root
    property bool immediate
    property string day
    property string clock
    property string startAddress
    property string endAddress
    property string pickupKm
    property string distanceKm
    property string fare
    property string euro
    property string server
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

        Row {
            width: parent.width
            spacing: Theme.paddingLarge

            RideMark {
                id: mark
                anchors.verticalCenter: parent.verticalCenter
                immediate: root.immediate
                day: root.day
                clock: root.clock
                highlighted: root.highlighted
            }

            Column {
                width: parent.width - mark.width - Theme.paddingLarge
                spacing: Theme.paddingSmall

                Item {
                    width: parent.width
                    height: Math.max(startLabel.implicitHeight, fareCol.height)

                    Label {
                        id: startLabel
                        anchors.left: parent.left
                        anchors.right: fareCol.left
                        anchors.rightMargin: Theme.paddingMedium
                        anchors.verticalCenter: parent.verticalCenter
                        text: root.startAddress
                        font.pixelSize: Theme.fontSizeMedium
                        color: root.highlighted ? Theme.highlightColor : Theme.primaryColor
                        truncationMode: TruncationMode.Fade
                    }

                    Column {
                        id: fareCol
                        anchors.right: parent.right
                        anchors.verticalCenter: parent.verticalCenter
                        spacing: 0

                        Label {
                            anchors.right: parent.right
                            text: root.fare
                            font.pixelSize: Theme.fontSizeSmall
                            color: root.highlighted ? Theme.highlightColor : Theme.primaryColor
                            horizontalAlignment: Text.AlignRight
                        }

                        Label {
                            anchors.right: parent.right
                            visible: root.euro !== ""
                            text: root.euro
                            font.pixelSize: Theme.fontSizeExtraSmall
                            color: root.highlighted ? Theme.secondaryHighlightColor : Theme.secondaryColor
                            horizontalAlignment: Text.AlignRight
                        }
                    }
                }

                Label {
                    width: parent.width
                    text: root.endAddress
                    font.pixelSize: Theme.fontSizeSmall
                    color: root.highlighted ? Theme.secondaryHighlightColor : Theme.secondaryColor
                    truncationMode: TruncationMode.Fade
                }
            }
        }

        Item {
            width: parent.width
            height: stats.height

            Row {
                id: stats
                spacing: Theme.paddingLarge

                RideStat {
                    value: root.pickupKm
                    caption: "Away"
                    highlighted: root.highlighted
                }

                RideStat {
                    value: root.distanceKm
                    caption: "Trip"
                    highlighted: root.highlighted
                }
            }

            Label {
                anchors.right: parent.right
                anchors.bottom: parent.bottom
                width: Math.max(0, parent.width - stats.width - Theme.paddingLarge)
                horizontalAlignment: Text.AlignRight
                text: root.server
                font.pixelSize: Theme.fontSizeExtraSmall
                color: root.highlighted ? Theme.secondaryHighlightColor : Theme.secondaryColor
                truncationMode: TruncationMode.Fade
            }
        }
    }
}
