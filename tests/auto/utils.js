// SPDX-FileCopyrightText: 2013-2023 Jolla Ltd.
// SPDX-FileCopyrightText: 2024-2025 Jolla Mobile Ltd
//
// SPDX-License-Identifier: BSD-3-Clause

.pragma library

function printChildren(item, depth) {
    for (var i = 0; i < item.children.length; ++i) {
        var space = function() {
            var str = ""
            for (var i = 0; i < depth; ++i)
                str += "-"
            return str + " "
        }()
        console.log(space + "child: "  + item.children[i])
        printChildren(item.children[i], depth + 1)
    }
}


function findChild(item, qmlType) {
    for (var i = 0; i < item.children.length; ++i) {
        if (item.children[i].toString().indexOf(qmlType) == 0)
            return item.children[i]

        var child = findChild(item.children[i], qmlType)
        if (child)
            return child
    }
}

function findChildData(item, qmlType) {
    for (var i = 0; i < item.data.length; ++i)
    {
        if (item.data[i].toString().indexOf(qmlType) == 0) {
            return item.data[i]
        }
    }
}

