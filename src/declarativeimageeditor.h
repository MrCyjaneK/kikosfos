/*
 * SPDX-FileCopyrightText: 2013-2023 Jolla Ltd.
 * SPDX-FileCopyrightText: 2024-2025 Jolla Mobile Ltd
 *
 * SPDX-License-Identifier: BSD-3-Clause
 */

#ifndef DECLARATIVEIMAGEEDITOR_H
#define DECLARATIVEIMAGEEDITOR_H

#include <QtQml>
#include <QQuickItem>
#include <QString>

class DeclarativeImageEditorPrivate;

class DeclarativeImageEditor : public QQuickItem
{
    Q_OBJECT
    Q_PROPERTY(QUrl source READ source WRITE setSource NOTIFY sourceChanged)
    Q_PROPERTY(QUrl target READ target WRITE setTarget NOTIFY targetChanged)
    Q_ENUMS(EditOperationType)

public:

    enum EditOperationType {
        None,
        Crop,
        Rotate,
        AdjustLevels,
    };

    explicit DeclarativeImageEditor(QQuickItem *parent = 0);
    virtual ~DeclarativeImageEditor();

    QUrl source() const;
    void setSource(const QUrl &source);

    QUrl target() const;
    void setTarget(const QUrl &target);

    Q_INVOKABLE void crop(const QSizeF &cropSize, const QSizeF &imageSize, const QPointF &position);
    Q_INVOKABLE void rotate(int rotation);
    Q_INVOKABLE void adjustLevels(double brightness, double contrast);

Q_SIGNALS:
    void cropped(bool success);
    void rotated(bool success);
    void levelsAdjusted(bool success);
    void sourceChanged();
    void targetChanged();

private Q_SLOTS:
    void cropResult(bool success, const QString &targetFile);
    void rotateResult(bool success, const QString &targetFile);
    void adjustLevelsResult(bool success, const QString &targetFile);

private:
    DeclarativeImageEditorPrivate *d_ptr;
    Q_DECLARE_PRIVATE(DeclarativeImageEditor)
};

QML_DECLARE_TYPE(DeclarativeImageEditor)

#endif // DECLARATIVEIMAGEEDITOR_H
