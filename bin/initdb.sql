
CREATE TABLE User (
    id  varchar(50) primary key unique not null,                --id
    User varchar(50)  not null,                                 --用户名
    Passwd varchar(50) not null,                                --密码
    Role varchar(10) not null default 'guest',                  --角色
    Total int not null default 0,                               --登录统计
    Host varchar(20),                                           --登录地址
    Clock time                                                  --更新时间
);
CREATE TABLE Auth (
    Token varchar(50) primary key unique not null,              --TOKEN
    Host varchar(50) not null,                                  --地址,
    User varchar(10) not null,                                  --用户
    Clock time not null default (datetime('now', 'localtime'))  --更新时间
, TokenAll varchar(500));


--初始化admin
insert into User(id,User,Passwd,Role) values('000000', 'admin', '123456', 'admin');
