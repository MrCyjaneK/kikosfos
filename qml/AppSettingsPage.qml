import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    id: page
    property string errorText
    property string placeText
    property bool ready

    onStatusChanged: {
        if (status === PageStatus.Active)
            load()
        else if (status === PageStatus.Inactive)
            kiko.gpsRelease()
    }

    function refreshPlace() {
        if (status !== PageStatus.Active)
            return
        var text = kiko.gpsPosition()
        if (text === "") {
            placeText = kiko.lastError()
            return
        }
        var parts = text.split("\n")
        placeText = parts[0] + ", " + parts[1]
        if (parts.length > 2 && parts[2] !== "")
            placeText += "  ±" + parts[2] + " m"
    }

    function load() {
        errorText = ""
        ready = false
        gpsBox.currentIndex = kiko.gpsDriver() === "mock" ? 1 : 0
        if (kiko.lastError() !== "")
            errorText = kiko.lastError()
        ready = true
        refreshPlace()
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: parent.width
            spacing: Theme.paddingLarge

            PageHeader { title: "Settings" }

            ComboBox {
                id: gpsBox
                width: parent.width
                label: "GPS"
                description: currentIndex === 1 ? "München Hauptbahnhof" : "Device location"
                menu: ContextMenu {
                    MenuItem { text: "GeoClue" }
                    MenuItem { text: "Mock" }
                }
                onCurrentIndexChanged: {
                    if (!page.ready)
                        return
                    var kind = currentIndex === 1 ? "mock" : "geoclue"
                    if (kiko.gpsSetDriver(kind) !== 0)
                        page.errorText = kiko.lastError()
                    else
                        page.errorText = ""
                    page.refreshPlace()
                }
            }

            Label {
                width: parent.width - 2 * Theme.horizontalPageMargin
                x: Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                font.pixelSize: Theme.fontSizeSmall
                text: page.placeText
            }

            Label {
                visible: page.errorText !== ""
                width: parent.width - 2 * Theme.horizontalPageMargin
                x: Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: page.errorText
            }
        }
    }

    Timer {
        interval: 1000
        repeat: true
        running: page.status === PageStatus.Active
        onTriggered: page.refreshPlace()
    }
}
