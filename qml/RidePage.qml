import QtQuick 2.6
import Sailfish.Silica 1.0
import "components"

Page {
    id: page
    property int coin: -1
    property string server
    property string startAddress
    property string endAddress
    property string pickupKm
    property string distanceKm
    property bool immediate
    property string whenDay
    property string whenClock
    property string startLat
    property string startLon
    property string fareLovelace
    property string baseLovelace
    property string perKmLovelace
    property string baseAmount
    property string baseEuro
    property string perKmAmount
    property string perKmEuro
    property string fareAmount
    property string fareEuro
    property bool euroTried
    property bool starting
    property string navigateText

    Component.onCompleted: refreshQuotes()

    function quoteParts(lovelace) {
        if (lovelace === "")
            return { amount: "", euro: "" }
        if (coin < 0)
            return { amount: lovelace, euro: "" }
        var parts = kiko.walletQuote(coin, lovelace, "lovelace").split("\n")
        if (parts.length > 3 && parts[3] !== "")
            return { amount: lovelace, euro: "" }
        return { amount: parts[0], euro: parts[2] }
    }

    function refreshQuotes() {
        var base = quoteParts(baseLovelace)
        var perKm = quoteParts(perKmLovelace)
        var fare = quoteParts(fareLovelace)
        baseAmount = base.amount
        baseEuro = base.euro
        perKmAmount = perKm.amount
        perKmEuro = perKm.euro
        fareAmount = fare.amount
        fareEuro = fare.euro
        if (!euroTried && coin >= 0 && fareEuro === "" && fareLovelace !== "" && fareLovelace !== "0") {
            euroTried = true
            kiko.walletWarmEuro(coin)
        }
    }

    function startRide() {
        if (starting)
            return
        starting = true
        navigateText = ""
        kiko.driveNavigate(startLat, startLon, startAddress)
    }

    Connections {
        target: kiko
        onEuroReady: {
            if (wallet === page.coin)
                page.refreshQuotes()
        }
        onNavigateFinished: {
            page.starting = false
            page.navigateText = message
        }
    }

    SilicaFlickable {
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: footer.top
        contentHeight: column.height + Theme.paddingLarge
        VerticalScrollDecorator {}

        Column {
            id: column
            width: parent.width
            spacing: Theme.paddingLarge

            PageHeader { title: "Ride" }

            Row {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                spacing: Theme.paddingLarge

                RideMark {
                    id: mark
                    immediate: page.immediate
                    day: page.whenDay
                    clock: page.whenClock
                }

                Column {
                    width: parent.width - mark.width - Theme.paddingLarge
                    spacing: Theme.paddingMedium

                    Label {
                        width: parent.width
                        wrapMode: Text.Wrap
                        text: page.startAddress
                        font.pixelSize: Theme.fontSizeLarge
                        color: Theme.primaryColor
                    }

                    Label {
                        width: parent.width
                        wrapMode: Text.Wrap
                        text: page.endAddress
                        color: Theme.secondaryColor
                    }
                }
            }

            Row {
                x: Theme.horizontalPageMargin
                spacing: Theme.paddingLarge

                RideStat {
                    value: page.pickupKm
                    caption: "Away"
                }

                RideStat {
                    value: page.distanceKm
                    caption: "Trip"
                }
            }

            Label {
                width: parent.width - 2 * Theme.horizontalPageMargin
                x: Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                text: page.server
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.secondaryColor
            }

            SectionHeader { text: "Price" }

            Column {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                spacing: Theme.paddingSmall

                Label {
                    id: captionMeasure
                    visible: false
                    text: "Per km"
                    font.pixelSize: Theme.fontSizeSmall
                }

                PriceLine {
                    caption: "Base"
                    captionWidth: captionMeasure.implicitWidth
                    amount: page.baseAmount
                    euro: page.baseEuro
                }

                PriceLine {
                    caption: "Per km"
                    captionWidth: captionMeasure.implicitWidth
                    amount: page.perKmAmount
                    euro: page.perKmEuro
                }

                PriceLine {
                    caption: "Trip"
                    captionWidth: captionMeasure.implicitWidth
                    amount: page.distanceKm + " km"
                }

                PriceLine {
                    caption: "Fare"
                    captionWidth: captionMeasure.implicitWidth
                    amount: page.fareAmount
                    euro: page.fareEuro
                }
            }
        }
    }

    Column {
        id: footer
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        spacing: Theme.paddingMedium

        BusyIndicator {
            anchors.horizontalCenter: parent.horizontalCenter
            running: page.starting
            visible: running
            width: Theme.itemSizeSmall
            height: width
        }

        Label {
            visible: page.navigateText !== ""
            width: parent.width - 2 * Theme.horizontalPageMargin
            x: Theme.horizontalPageMargin
            wrapMode: Text.Wrap
            horizontalAlignment: Text.AlignHCenter
            color: Theme.highlightColor
            font.pixelSize: Theme.fontSizeSmall
            text: page.navigateText
        }

        Button {
            anchors.horizontalCenter: parent.horizontalCenter
            text: "Start"
            enabled: !page.starting
            onClicked: page.startRide()
        }

        Item {
            width: 1
            height: Theme.paddingLarge
        }
    }
}
