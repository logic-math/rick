# 学习使命 — loop_8

主题：https://arxiv.org/html/2609.28589v1
论文：OneTrans-V2: Unifying Retrieval, Pre-rank, and Fine-rank with One Transformer in Industrial Recommender

## 学习目标（human 原话）

1. 理解这篇论文在做什么、原理是什么；将其用在实际工作中，找到原创点。
2. 向同事讲清 OneTrans-V2 如何用一个 Transformer 同时服务三个阶段的输入/输出/训练差异。

## 考核标准（human 原话）

> 向同事讲清 OneTrans-V2 如何用一个 Transformer 同时服务三个阶段的输入/输出/训练差异。

（即：不看原文，能独立讲清「一个 Transformer 如何同时服务 召回/粗排/精排 三个阶段」在**输入**、**输出**、**训练**三方面的差异与统一方式。）

## 起点（human 原话）

> 我是机器学习架构-推理引擎工程师，对推荐系统本身是很熟悉的。我只是对这篇论文如何做的不了解。

教学定位：**不教推荐系统基础概念**，聚焦 OneTrans-V2 的「如何做」——统一架构的输入/输出/训练差异、一个 Transformer 服务三阶段的机制、以及可迁移到实际工作（推理引擎视角）的原创点。
