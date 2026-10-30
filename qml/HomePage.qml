import QtQuick 2.6
import Sailfish.Silica 1.0
import "components"

Page {
    id: page
    property string latText
    property string lonText
    property string accText
    property string placeText
    property string registerText
    property string hint
    property int count: 0
    property int activeCount: 0
    property int stamp: 0
    property int activeStamp: 0

    Component.onCompleted: sync()

    onStatusChanged: {
        if (status === PageStatus.Active)
            reloadActive()
    }

    function refreshPlace() {
        var text = kiko.gpsPosition()
        if (text === "") {
            latText = ""
            lonText = ""
            accText = ""
            var err = kiko.lastError()
            placeText = err !== "" ? err : "Establishing GPS"
            return
        }
        var parts = text.split("\n")
        latText = parts[0]
        lonText = parts.length > 1 ? parts[1] : ""
        accText = parts.length > 2 ? parts[2] : ""
        placeText = ""
    }

    function sync() {
        refreshPlace()
        reloadActive()
        kiko.driveRegister()
    }

    function reloadActive() {
        var n = kiko.driveActiveCount()
        activeCount = n < 0 ? 0 : n
        activeStamp++
    }

    function openRide(ride) {
        pageStack.push(Qt.resolvedUrl("RidePage.qml"), {
            coin: ride.coin,
            server: ride.server,
            startAddress: ride.start_address,
            endAddress: ride.end_address,
            pickupKm: ride.pickup_km,
            distanceKm: ride.distance_km,
            immediate: ride.immediate === 1,
            whenDay: ride.when_day,
            whenClock: ride.when_clock,
            rideId: ride.id,
            startLat: ride.start_lat,
            startLon: ride.start_lon,
            endLat: ride.end_lat,
            endLon: ride.end_lon,
            fareLovelace: ride.fare_lovelace,
            baseLovelace: ride.base_lovelace,
            perKmLovelace: ride.per_km_lovelace,
            rideStatus: ride.status
        })
    }

    function quoteParts(coin, lovelace) {
        if (lovelace === "")
            return { amount: "", euro: "" }
        if (coin < 0)
            return { amount: lovelace, euro: "" }
        var parts = kiko.walletQuote(coin, lovelace, "lovelace").split("\n")
        if (parts.length > 3 && parts[3] !== "")
            return { amount: lovelace, euro: "" }
        return { amount: parts[0], euro: parts[2] }
    }

    function warmEuros() {
        var seen = {}
        for (var i = 0; i < count; i++) {
            var coin = kiko.driveCoin(i)
            if (coin >= 0 && !seen[coin]) {
                seen[coin] = true
                kiko.walletWarmEuro(coin)
            }
        }
    }

    Connections {
        target: kiko
        onDriveRegistered: {
            page.registerText = message
            page.reloadActive()
            kiko.driveRefresh(5)
        }
        onCashOutFinished: page.reloadActive()
        onClaimFinished: {
            page.reloadActive()
            if (message === "") {
                var n = kiko.driveCount()
                page.count = n < 0 ? 0 : n
                page.stamp++
                kiko.driveRefresh(5)
            }
        }
        onDrivesReady: {
            var n = kiko.driveCount()
            page.count = n < 0 ? 0 : n
            if (message !== "")
                page.hint = message
            else if (page.count === 0)
                page.hint = "No rides"
            else
                page.hint = ""
            page.stamp++
            page.warmEuros()
        }
        onEuroReady: page.stamp++
    }

    SilicaListView {
        id: list
        anchors.fill: parent
        model: page.count

        PullDownMenu {
            MenuItem {
                text: "Refresh"
                onClicked: page.sync()
            }
            MenuItem {
                text: "Wallets"
                onClicked: pageStack.push(Qt.resolvedUrl("WalletsPage.qml"))
            }
            MenuItem {
                text: "Servers"
                onClicked: pageStack.push(Qt.resolvedUrl("ServersPage.qml"))
            }
            MenuItem {
                text: "Profile"
                onClicked: pageStack.push(Qt.resolvedUrl("ProfilePage.qml"))
            }
            MenuItem {
                text: "Settings"
                onClicked: pageStack.push(Qt.resolvedUrl("AppSettingsPage.qml"))
            }
        }

        header: Column {
            width: list.width

            PageHeader { title: "Kiko" }

            GpsFix {
                width: parent.width
                lat: page.latText
                lon: page.lonText
                accuracy: page.accText
                message: page.placeText
            }

            Item {
                width: parent.width
                height: page.registerText === "" ? 0 : regLabel.implicitHeight + Theme.paddingMedium

                Label {
                    id: regLabel
                    x: Theme.horizontalPageMargin
                    width: parent.width - 2 * Theme.horizontalPageMargin
                    wrapMode: Text.Wrap
                    color: Theme.highlightColor
                    font.pixelSize: Theme.fontSizeSmall
                    text: page.registerText
                }
            }

            Column {
                width: parent.width
                visible: page.activeCount > 0
                height: visible ? implicitHeight : 0
                spacing: Theme.paddingSmall

                Label {
                    x: Theme.horizontalPageMargin
                    width: parent.width - 2 * x
                    text: "Your ride"
                    font.pixelSize: Theme.fontSizeMedium
                    color: Theme.highlightColor
                }

                Repeater {
                    model: page.activeCount

                    Item {
                        id: activeRow
                        width: list.width
                        height: activePress.height
                        property var ride: ({})
                        property string fare
                        property string euro

                        function load() {
                            var text = kiko.driveActive(index)
                            if (text === "")
                                return
                            var parsed = JSON.parse(text)
                            ride = parsed
                            var parts = page.quoteParts(parsed.coin, parsed.fare_lovelace)
                            fare = parts.amount
                            euro = parts.euro
                        }

                        Component.onCompleted: load()

                        Connections {
                            target: page
                            onActiveStampChanged: activeRow.load()
                            onStampChanged: activeRow.load()
                        }

                        MouseArea {
                            id: activePress
                            width: parent.width
                            height: activeCard.height + Theme.paddingMedium
                            onClicked: page.openRide(activeRow.ride)

                            RideCard {
                                id: activeCard
                                x: Theme.horizontalPageMargin
                                y: Theme.paddingSmall
                                width: parent.width - 2 * Theme.horizontalPageMargin
                                highlighted: activePress.pressed
                                immediate: activeRow.ride.immediate === 1
                                day: activeRow.ride.when_day || ""
                                clock: activeRow.ride.when_clock || ""
                                startAddress: activeRow.ride.start_address || ""
                                endAddress: activeRow.ride.end_address || ""
                                pickupKm: activeRow.ride.pickup_km || ""
                                distanceKm: activeRow.ride.distance_km || ""
                                fare: activeRow.fare
                                euro: activeRow.euro
                                server: activeRow.ride.server || ""
                            }
                        }
                    }
                }
            }
        }

        ViewPlaceholder {
            enabled: page.count === 0 && page.activeCount === 0 && page.hint !== ""
            text: page.hint
        }

        delegate: Column {
            id: row
            width: ListView.view.width
            spacing: 0
            property int coin: -1
            property string server
            property string startAddress
            property string endAddress
            property string pickupKm
            property string distanceKm
            property bool immediate
            property string whenDay
            property string whenClock
            property string rideId
            property string startLat
            property string startLon
            property string endLat
            property string endLon
            property bool showSection
            property string fareLovelace
            property string baseLovelace
            property string perKmLovelace
            property string fare
            property string euro

            function reloadRow() {
                coin = kiko.driveCoin(index)
                server = kiko.driveServer(index)
                startAddress = kiko.driveStartAddress(index)
                endAddress = kiko.driveEndAddress(index)
                pickupKm = kiko.drivePickupKm(index)
                distanceKm = kiko.driveDistanceKm(index)
                immediate = kiko.driveImmediate(index) === 1
                whenDay = kiko.driveWhenDay(index)
                whenClock = kiko.driveWhenClock(index)
                rideId = kiko.driveId(index)
                startLat = kiko.driveStartLat(index)
                startLon = kiko.driveStartLon(index)
                endLat = kiko.driveEndLat(index)
                endLon = kiko.driveEndLon(index)
                showSection = index === 0 || kiko.driveImmediate(index - 1) !== kiko.driveImmediate(index)
                fareLovelace = kiko.driveFareLovelace(index)
                baseLovelace = kiko.driveBaseLovelace(index)
                perKmLovelace = kiko.drivePerKmLovelace(index)
                var parts = page.quoteParts(coin, fareLovelace)
                fare = parts.amount
                euro = parts.euro
            }

            Component.onCompleted: reloadRow()

            Connections {
                target: page
                onStampChanged: row.reloadRow()
            }

            Item {
                width: parent.width
                height: row.showSection ? groupLabel.implicitHeight + Theme.paddingSmall : 0
                visible: row.showSection

                Label {
                    id: groupLabel
                    x: Theme.horizontalPageMargin
                    width: parent.width - 2 * x
                    anchors.bottom: parent.bottom
                    text: row.immediate ? "Now" : "Later"
                    font.pixelSize: Theme.fontSizeExtraSmall
                    color: Theme.highlightColor
                }
            }

            MouseArea {
                id: press
                width: parent.width
                height: card.height + Theme.paddingMedium
                onClicked: pageStack.push(Qt.resolvedUrl("RidePage.qml"), {
                    coin: row.coin,
                    server: row.server,
                    startAddress: row.startAddress,
                    endAddress: row.endAddress,
                    pickupKm: row.pickupKm,
                    distanceKm: row.distanceKm,
                    immediate: row.immediate,
                    whenDay: row.whenDay,
                    whenClock: row.whenClock,
                    rideId: row.rideId,
                    startLat: row.startLat,
                    startLon: row.startLon,
                    endLat: row.endLat,
                    endLon: row.endLon,
                    fareLovelace: row.fareLovelace,
                    baseLovelace: row.baseLovelace,
                    perKmLovelace: row.perKmLovelace
                })

                RideCard {
                    id: card
                    x: Theme.horizontalPageMargin
                    y: Theme.paddingSmall
                    width: parent.width - 2 * Theme.horizontalPageMargin
                    highlighted: press.pressed
                    immediate: row.immediate
                    day: row.whenDay
                    clock: row.whenClock
                    startAddress: row.startAddress
                    endAddress: row.endAddress
                    pickupKm: row.pickupKm
                    distanceKm: row.distanceKm
                    fare: row.fare
                    euro: row.euro
                    server: row.server
                }
            }
        }
    }

    Timer {
        interval: 1000
        repeat: true
        running: page.status === PageStatus.Active
        onTriggered: page.refreshPlace()
    }

    Timer {
        interval: 5 * 60 * 1000
        repeat: true
        running: true
        onTriggered: page.sync()
    }
}
