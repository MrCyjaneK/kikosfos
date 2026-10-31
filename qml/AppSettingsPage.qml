import QtQuick 2.6
import Sailfish.Silica 1.0
import "components"

Page {
    id: page
    property string errorText
    property string latText
    property string lonText
    property string accText
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
            latText = ""
            lonText = ""
            accText = ""
            placeText = kiko.lastError()
            return
        }
        var parts = text.split("\n")
        latText = parts[0]
        lonText = parts.length > 1 ? parts[1] : ""
        accText = parts.length > 2 ? parts[2] : ""
        placeText = ""
    }

    function load() {
        errorText = ""
        ready = false
        gpsBox.currentIndex = kiko.gpsDriver() === "mock" ? 1 : 0
        var km = kiko.driveRange()
        if (!(km >= 1))
            km = 5
        rangeSlider.value = km
        if (kiko.lastError() !== "")
            errorText = kiko.lastError()
        ready = true
        refreshPlace()
    }

    function saveRange() {
        var km = Math.round(rangeSlider.value)
        if (kiko.driveSetRange(km) !== 0)
            errorText = kiko.lastError()
        else
            errorText = ""
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

            Slider {
                id: rangeSlider
                width: parent.width
                minimumValue: 1
                maximumValue: 50
                stepSize: 1
                label: "Look for offers"
                valueText: Math.round(value) + " km"
                onValueChanged: {
                    if (!page.ready || down)
                        return
                    page.saveRange()
                }
                onDownChanged: {
                    if (down || !page.ready)
                        return
                    page.saveRange()
                }
            }

            GpsFix {
                width: parent.width
                lat: page.latText
                lon: page.lonText
                accuracy: page.accText
                message: page.placeText
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
