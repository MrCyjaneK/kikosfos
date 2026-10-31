import QtQuick 2.6
import Sailfish.Silica 1.0
import "components"

Page {
    id: page
    property int index: -1
    property string host
    property string errorText
    property var chains: []

    Component.onCompleted: load()

    function load() {
        if (index < 0)
            return
        hostField.text = host
        var text = kiko.serverChains(index)
        if (text === "") {
            chains = []
            if (kiko.lastError() !== "")
                errorText = kiko.lastError()
            return
        }
        try {
            chains = JSON.parse(text)
        } catch (e) {
            chains = []
            errorText = "Could not read this server"
        }
    }

    function save() {
        errorText = ""
        if (index < 0) {
            if (kiko.serverAdd(hostField.text) !== 0)
                errorText = kiko.lastError()
            else
                pageStack.pop()
            return
        }
        var drafts = []
        for (var i = 0; i < chainRepeater.count; i++) {
            var row = chainRepeater.itemAt(i).feePayload()
            if (row)
                drafts.push(row)
        }
        if (kiko.serverUpdate(index, hostField.text) !== 0) {
            errorText = kiko.lastError()
            return
        }
        var text = kiko.serverChains(index)
        var offered = []
        try {
            offered = JSON.parse(text)
        } catch (e) {
            errorText = kiko.lastError() !== "" ? kiko.lastError() : "Could not read this server"
            return
        }
        var payload = []
        for (var c = 0; c < offered.length; c++) {
            for (var d = 0; d < drafts.length; d++) {
                if (drafts[d].currency === offered[c].currency)
                    payload.push(drafts[d])
            }
        }
        if (kiko.serverSaveChains(index, JSON.stringify(payload)) !== 0) {
            errorText = kiko.lastError()
            return
        }
        pageStack.pop()
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: parent.width
            spacing: Theme.paddingLarge

            PageHeader { title: page.index < 0 ? "Add server" : "Server" }

            Label {
                visible: page.errorText !== ""
                width: parent.width - 2 * Theme.horizontalPageMargin
                x: Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: page.errorText
            }

            TextField {
                id: hostField
                width: parent.width
                label: "Host"
                placeholderText: "example.com"
                inputMethodHints: Qt.ImhNoAutoUppercase | Qt.ImhUrlCharactersOnly
            }

            Repeater {
                id: chainRepeater
                model: page.index < 0 ? 0 : page.chains.length

                ChainPrice {
                    currency: page.chains[index].currency
                    coin: page.chains[index].coin
                    coinName: page.chains[index].name
                    whole: page.chains[index].whole
                    atom: page.chains[index].atom
                    ticker: page.chains[index].ticker
                    hasWallet: page.chains[index].wallet
                    baseFee: page.chains[index].base_fee
                    perKmFee: page.chains[index].fee_per_km
                }
            }

            Button {
                anchors.horizontalCenter: parent.horizontalCenter
                text: page.index < 0 ? "Add" : "Save"
                onClicked: page.save()
            }

            Button {
                visible: page.index >= 0
                anchors.horizontalCenter: parent.horizontalCenter
                text: "Delete"
                onClicked: {
                    if (kiko.serverDelete(page.index) === 0)
                        pageStack.pop()
                    else
                        page.errorText = kiko.lastError()
                }
            }
        }
    }
}
