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
        var refreshError = ""
        if (kiko.serverRefresh() !== 0)
            refreshError = kiko.lastError()
        var n = kiko.serverCount()
        if (n < 0) {
            errorText = kiko.lastError()
            count = 0
        } else {
            errorText = refreshError
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
            var text = kiko.serverChains(i)
            var chains = []
            try {
                chains = JSON.parse(text)
            } catch (e) {
                chains = []
            }
            for (var j = 0; j < chains.length; j++) {
                var coin = chains[j].coin
                if (chains[j].wallet && coin >= 0 && !seen[coin]) {
                    seen[coin] = true
                    asked = true
                    kiko.walletWarmEuro(coin)
                }
            }
        }
        if (asked)
            euroTried = true
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
            property string chainsText

            function reloadRow() {
                host = kiko.serverHost(index)
                chainsText = kiko.serverChains(index)
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
                    chainsText: row.chainsText
                    stamp: page.stamp
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
