import QtQuick 2.6
import Sailfish.Silica 1.0

Rectangle {
    id: root
    property string host
    property string chainsText
    property int stamp
    property bool highlighted
    property var rows: []

    radius: Theme.paddingSmall
    color: root.highlighted
           ? Theme.highlightBackgroundColor
           : Theme.rgba(Theme.primaryColor, 0.06)
    implicitHeight: body.height + 2 * Theme.paddingMedium
    height: implicitHeight

    function quote(coin, atoms, atom) {
        if (atoms === "")
            return { amount: "", euro: "" }
        if (coin < 0)
            return { amount: atoms, euro: "" }
        var parts = kiko.walletQuote(coin, atoms, atom).split("\n")
        if (parts.length > 3 && parts[3] !== "")
            return { amount: atoms, euro: "" }
        return { amount: parts[0], euro: parts[2] }
    }

    function reload() {
        var parsed = []
        if (chainsText !== "") {
            try {
                parsed = JSON.parse(chainsText)
            } catch (e) {
                parsed = []
            }
        }
        var next = []
        for (var i = 0; i < parsed.length; i++) {
            var chain = parsed[i]
            var priced = chain.wallet && (chain.base_fee !== "" || chain.fee_per_km !== "")
            var base = priced ? quote(chain.coin, chain.base_fee, chain.atom) : { amount: "", euro: "" }
            var perKm = priced ? quote(chain.coin, chain.fee_per_km, chain.atom) : { amount: "", euro: "" }
            next.push({
                name: chain.name || chain.currency,
                wallet: chain.wallet,
                hasPrice: priced,
                baseAmount: base.amount,
                baseEuro: base.euro,
                perKmAmount: perKm.amount,
                perKmEuro: perKm.euro
            })
        }
        rows = next
    }

    onChainsTextChanged: reload()
    onStampChanged: reload()

    Column {
        id: body
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.margins: Theme.paddingMedium
        spacing: Theme.paddingMedium

        Label {
            width: parent.width
            text: root.host
            font.pixelSize: Theme.fontSizeMedium
            color: root.highlighted ? Theme.highlightColor : Theme.primaryColor
            truncationMode: TruncationMode.Fade
        }

        Label {
            id: captionMeasure
            visible: false
            text: "Per km"
            font.pixelSize: Theme.fontSizeSmall
        }

        Repeater {
            model: root.rows.length

            Column {
                width: body.width
                spacing: Theme.paddingSmall
                property var row: root.rows[index]

                Label {
                    width: parent.width
                    text: row.name
                    font.pixelSize: Theme.fontSizeSmall
                    color: root.highlighted ? Theme.highlightColor : Theme.primaryColor
                }

                Label {
                    visible: row && !row.wallet
                    width: parent.width
                    text: "No wallet"
                    font.pixelSize: Theme.fontSizeSmall
                    color: root.highlighted ? Theme.secondaryHighlightColor : Theme.secondaryColor
                }

                Label {
                    visible: row && row.wallet && !row.hasPrice
                    width: parent.width
                    text: "No price"
                    font.pixelSize: Theme.fontSizeSmall
                    color: root.highlighted ? Theme.secondaryHighlightColor : Theme.secondaryColor
                }

                PriceLine {
                    visible: row && row.hasPrice
                    caption: "Base"
                    captionWidth: captionMeasure.implicitWidth
                    amount: row ? row.baseAmount : ""
                    euro: row ? row.baseEuro : ""
                    highlighted: root.highlighted
                }

                PriceLine {
                    visible: row && row.hasPrice
                    caption: "Per km"
                    captionWidth: captionMeasure.implicitWidth
                    amount: row ? row.perKmAmount : ""
                    euro: row ? row.perKmEuro : ""
                    highlighted: root.highlighted
                }
            }
        }
    }
}
