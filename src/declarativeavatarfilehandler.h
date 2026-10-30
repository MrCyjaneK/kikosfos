/*
 * SPDX-FileCopyrightText: 2014-2023 Jolla Ltd.
 * SPDX-FileCopyrightText: 2024-2025 Jolla Mobile Ltd
 *
 * SPDX-License-Identifier: BSD-3-Clause
 */

#ifndef DECLARATIVEAVATARFILEHANDLER_H
#define DECLARATIVEAVATARFILEHANDLER_H

#include <QObject>
#include <QUrl>
#include <QQmlEngine>

class DeclarativeAvatarFileHandler: public QObject
{
    Q_OBJECT
public:
    DeclarativeAvatarFileHandler(QObject *parent = 0);

    Q_INVOKABLE QString createNewAvatarFileName(const QString &base);

    static QObject *api_factory(QQmlEngine *, QJSEngine *);
};

#endif // DECLARATIVEAVATARFILEHANDLER_H
