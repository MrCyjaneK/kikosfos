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
    property string rideId
    property string whenDay
    property string whenClock
    property string startLat
    property string startLon
    property string endLat
    property string endLon
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
    property bool working
    property bool claimed
    property bool paid
    property string rideStatus
    property string notice

    Component.onCompleted: {
        refreshQuotes()
        claimed = kiko.driveClaimed(server, rideId) === 1
        paid = kiko.drivePaid(server, rideId) === 1
        if (rideStatus === "" && paid)
            rideStatus = "paid"
        else if (rideStatus === "" && claimed)
            rideStatus = "accepted"
        if (paid)
            notice = rideStatus === "paid_out" ? "Paid out" : "Paid"
    }

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

    function openNav(lat, lon, title) {
        if (working)
            return
        working = true
        notice = page.rideStatus === "paid_out" ? "Paid out" : (paid ? "Paid" : "")
        kiko.driveNavigate(lat, lon, title)
    }

    function claimRide() {
        if (working || claimed)
            return
        working = true
        notice = ""
        kiko.driveClaim(server, rideId, pickupKm)
    }

    function checkPayment() {
        if (working || !claimed || paid)
            return
        working = true
        notice = ""
        kiko.driveVerify(server, rideId)
    }

    function cashOut() {
        if (working || !paid || rideStatus === "paid_out")
            return
        working = true
        notice = ""
        kiko.driveCashOut(server, rideId)
    }

    Connections {
        target: kiko
        onEuroReady: {
            if (wallet === page.coin)
                page.refreshQuotes()
        }
        onNavigateFinished: {
            page.working = false
            if (message !== "")
                page.notice = message
            else if (page.paid)
                page.notice = "Paid"
        }
        onClaimFinished: {
            page.working = false
            if (message === "") {
                page.claimed = true
                page.rideStatus = "accepted"
            } else
                page.notice = message
        }
        onDriveVerified: {
            page.working = false
            if (message !== "") {
                page.notice = message
                return
            }
            page.paid = seen === 1
            if (page.paid)
                page.rideStatus = "paid"
            page.notice = page.paid ? "Paid" : "Payment not seen yet"
        }
        onCashOutFinished: {
            page.working = false
            if (message === "") {
                page.rideStatus = "paid_out"
                page.notice = "Paid out"
            } else
                page.notice = message
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
            running: page.working
            visible: running
            width: Theme.itemSizeSmall
            height: width
        }

        Label {
            visible: page.notice !== ""
            width: parent.width - 2 * Theme.horizontalPageMargin
            x: Theme.horizontalPageMargin
            wrapMode: Text.Wrap
            horizontalAlignment: Text.AlignHCenter
            color: page.notice === "Paid" || page.notice === "Paid out" ? Theme.secondaryColor : Theme.highlightColor
            font.pixelSize: Theme.fontSizeSmall
            text: page.notice
        }

        Button {
            anchors.horizontalCenter: parent.horizontalCenter
            visible: !page.claimed
            text: "Claim"
            enabled: !page.working
            onClicked: page.claimRide()
        }

        Button {
            anchors.horizontalCenter: parent.horizontalCenter
            text: "To pickup"
            enabled: !page.working
            onClicked: page.openNav(startLat, startLon, startAddress)
        }

        Button {
            anchors.horizontalCenter: parent.horizontalCenter
            visible: page.claimed && !page.paid
            text: "Check payment"
            enabled: !page.working
            onClicked: page.checkPayment()
        }

        Button {
            anchors.horizontalCenter: parent.horizontalCenter
            text: "To destination"
            enabled: !page.working
            onClicked: page.openNav(endLat, endLon, endAddress)
        }

        Button {
            anchors.horizontalCenter: parent.horizontalCenter
            visible: page.paid && page.rideStatus !== "paid_out"
            text: "Cash out"
            enabled: !page.working
            onClicked: page.cashOut()
        }

        Item {
            width: 1
            height: Theme.paddingLarge
        }
    }
}
