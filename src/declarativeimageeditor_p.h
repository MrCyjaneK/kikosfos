/*
 * SPDX-FileCopyrightText: 2013-2023 Jolla Ltd.
 * SPDX-FileCopyrightText: 2024-2025 Jolla Mobile Ltd
 *
 * SPDX-License-Identifier: BSD-3-Clause
 */

#ifndef DECLARATIVEIMAGEEDITOR_PRIVATE_H
#define DECLARATIVEIMAGEEDITOR_PRIVATE_H

#include <QObject>
#include <QUrl>
#include <QDir>

class QString;
class QSizeF;
class QPointF;

class DeclarativeImageEditorPrivate : public QObject
{
    Q_OBJECT
public:
    explicit DeclarativeImageEditorPrivate(QObject *parent = 0);
    virtual ~DeclarativeImageEditorPrivate();

    QString uniqueFilePath(const QString &sourceFilePath, const QString &path = QDir::tempPath());
    QString save(QImage &image, const QString &source, const QString &target, const QByteArray &format = QByteArray());

    // Member variables
    QUrl m_source;
    QUrl m_target;

public Q_SLOTS:
    void rotate(const QString &source, const QString &target, int rotation);
    void crop(const QString &source, const QString &target, const QSizeF &cropSize,
              const QSizeF &imageSize, const QPointF &position);
    void adjustLevels(const QString &source, const QString &target, double brightness, double contrast);

Q_SIGNALS:
    void cropped(bool success, const QString &targetFile = QString());
    void rotated(bool success, const QString &targetFile = QString());
    void levelsAdjusted(bool success, const QString &targetFile = QString());
};

#endif // DECLARATIVEIMAGEEDITOR_PRIVATE_H
