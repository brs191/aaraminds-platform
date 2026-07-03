# Cost scenario — force spoke egress through a new Azure Firewall

You are forecasting the monthly cost impact of a proposed network change. Region: **East US**.

## Current state

A hub-and-spoke estate. Three spokes currently egress **directly to the internet** (their route tables send `0.0.0.0/0 → Internet`; no firewall in the path). There is no Azure Firewall today.

## Proposed change

Deploy one **Azure Firewall (Standard)** in the hub, give it a public IP, and change every spoke's default route to `0.0.0.0/0 → VirtualAppliance (the firewall)` so all spoke egress is inspected.

## Traffic basis

VNet flow logs + Traffic Analytics over the last 30 days show total spoke egress to the internet of **40–60 TB/month** (p50 ≈ 45 TB, p90 ≈ 58 TB).

## Ask

Forecast the **monthly cost delta** of this change. Show your working: what is fixed vs variable, where the money actually goes, and what you are uncertain about.
