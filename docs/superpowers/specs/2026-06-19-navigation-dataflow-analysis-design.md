# Navigation & Data Flow Analysis Design

> **Design doc for mxgraph-based navigation tree + page data container hierarchy + theme/appearance analysis.**

## Problem

Developers struggle to see:
1. Which pages are reachable from navigation vs orphaned
2. Data container hierarchy inside pages with context variables per nesting level
3. End-to-end entity → data container → page → navigation data flow
4. Theme/appearance + conditional visibility rules on widgets

## Solution

Add 2 new mxgraph IndexAdapters (NavigationAdapter, DataContainerAdapter) + enhance 2 existing adapters (WidgetInstanceAdapter, PageRefAdapter) + add graphcatalog readers + single `mxcli analyze` CLI command.

## Architecture

See inline design in session transcript - SOLID, SRP per adapter, ISP per reader, DIP per adapter→EventSink.
