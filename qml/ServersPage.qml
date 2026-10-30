import QtQuick 2.6
import Sailfish.Silica 1.0
import "components"

Page {
    id: page
    property int count: 0
    property string errorText
    property int stamp: 0
    property bool euroTried

    onStatusChanged: {
        if (status === PageStatus.Active)
            reload()
    }

    function reload() {
        var n = kiko.serverCount()
        if (n < 0) {
            errorText = kiko.lastError()
            count = 0
        } else {
            errorText = ""
            count = n
        }
        stamp++
        warmEuros()
    }

    function warmEuros() {
        if (euroTried)
            return
        var seen = {}
        var asked = false
        for (var i = 0; i < count; i++) {
            var coin = kiko.serverCoin(i)
            if (coin >= 0 && !seen[coin]) {
                seen[coin] = true
                asked = true
                kiko.walletWarmEuro(coin)
            }
        }
        if (asked)
            euroTried = true
    }

    function quoteParts(coin, lovelace) {
        if (lovelace === "")
            return { amount: "", euro: "" }
        if (coin < 0)
            return { amount: lovelace, euro: "" }
        var parts = kiko.walletQuote(coin, lovelace, kiko.walletAtom(coin)).split("\n")
        if (parts.length > 3 && parts[3] !== "")
            return { amount: lovelace, euro: "" }
        return { amount: parts[0], euro: parts[2] }
    }

    Connections {
        target: kiko
        onEuroReady: page.stamp++
    }

    SilicaListView {
        anchors.fill: parent
        model: page.count

        PullDownMenu {
            MenuItem {
                text: "Add"
                onClicked: pageStack.push(Qt.resolvedUrl("ServerPage.qml"), { index: -1 })
            }
            MenuItem {
                text: "Set all prices"
                enabled: page.count > 0
                onClicked: pageStack.push(Qt.resolvedUrl("PricesPage.qml"))
            }
        }

        header: PageHeader { title: "Servers" }

        ViewPlaceholder {
            enabled: page.count === 0 && page.errorText === ""
            text: "No servers"
            hintText: "Pull down to add a server"
        }

        delegate: Item {
            id: row
            width: ListView.view.width
            height: serverCard.height + Theme.paddingMedium
            property string host
            property string status
            property bool compatible
            property bool hasPrice
            property string baseAmount
            property string baseEuro
            property string perKmAmount
            property string perKmEuro

            function reloadRow() {
                host = kiko.serverHost(index)
                compatible = kiko.serverCompatible(index) === 1
                status = compatible ? kiko.serverWalletName(index) : "Incompatible"
                var coin = kiko.serverCoin(index)
                var base = kiko.serverFeeBase(index)
                var perKm = kiko.serverFeePerKm(index)
                hasPrice = base !== "" || perKm !== ""
                var baseParts = page.quoteParts(coin, base)
                var perParts = page.quoteParts(coin, perKm)
                baseAmount = baseParts.amount
                baseEuro = baseParts.euro
                perKmAmount = perParts.amount
                perKmEuro = perParts.euro
            }

            Component.onCompleted: reloadRow()

            Connections {
                target: page
                onStampChanged: row.reloadRow()
            }

            MouseArea {
                id: press
                anchors.fill: parent
                onClicked: pageStack.push(Qt.resolvedUrl("ServerPage.qml"), {
                    index: index,
                    host: row.host
                })

                ServerCard {
                    id: serverCard
                    x: Theme.horizontalPageMargin
                    y: Theme.paddingSmall
                    width: parent.width - 2 * Theme.horizontalPageMargin
                    highlighted: press.pressed
                    host: row.host
                    status: row.status
                    compatible: row.compatible
                    hasPrice: row.hasPrice
                    baseAmount: row.baseAmount
                    baseEuro: row.baseEuro
                    perKmAmount: row.perKmAmount
                    perKmEuro: row.perKmEuro
                }
            }
        }
    }

    Label {
        visible: page.errorText !== ""
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        anchors.margins: Theme.horizontalPageMargin
        wrapMode: Text.Wrap
        color: Theme.highlightColor
        text: page.errorText
    }
}
