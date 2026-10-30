#pragma once

#include <QObject>

class KikoApi : public QObject
{
    Q_OBJECT
public:
    explicit KikoApi(QObject *parent = nullptr);

    Q_INVOKABLE QString hello() const;
};
