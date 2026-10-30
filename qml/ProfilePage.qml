import QtQuick 2.6
import Sailfish.Silica 1.0

Page {
    id: page
    property string errorText

    onStatusChanged: {
        if (status === PageStatus.Active)
            load()
    }

    function load() {
        if (!brandField)
            return
        errorText = ""
        brandField.text = kiko.profileBrand()
        if (kiko.lastError() !== "") {
            errorText = kiko.lastError()
            return
        }
        modelField.text = kiko.profileModel()
        colorField.text = kiko.profileColor()
        plateField.text = kiko.profilePlate()
        yearField.text = kiko.profileYear()
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: parent.width
            spacing: Theme.paddingLarge

            PageHeader { title: "Profile" }

            Label {
                visible: page.errorText !== ""
                width: parent.width - 2 * Theme.horizontalPageMargin
                x: Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: page.errorText
            }

            TextField {
                id: brandField
                width: parent.width
                label: "Brand"
            }

            TextField {
                id: modelField
                width: parent.width
                label: "Model"
            }

            TextField {
                id: colorField
                width: parent.width
                label: "Color"
            }

            TextField {
                id: plateField
                width: parent.width
                label: "License plate"
            }

            TextField {
                id: yearField
                width: parent.width
                label: "Year"
                inputMethodHints: Qt.ImhDigitsOnly
            }

            Button {
                anchors.horizontalCenter: parent.horizontalCenter
                text: "Save"
                onClicked: {
                    if (kiko.profileSaveCar(brandField.text, modelField.text, colorField.text, plateField.text, yearField.text) === 0)
                        page.errorText = ""
                    else
                        page.errorText = kiko.lastError()
                }
            }
        }
    }
}
