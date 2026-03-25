# Glossary

Mendix-specific terms and concepts for developers who are new to the Mendix platform.

## A

**Association**
A relationship between two entities. Analogous to a foreign key in relational databases. Mendix supports `Reference` (one-to-many or one-to-one) and `ReferenceSet` (many-to-many) types.

**Attribute**
A property of an entity. Equivalent to a column in a relational database table. Types include String, Integer, Long, Decimal, Boolean, DateTime, AutoNumber, Binary, Enumeration, and HashedString.

**Access Rule**
A security rule that controls which module roles can create, read, write, or delete instances of an entity, and which attributes they can access.

## B

**BSON**
Binary JSON. The serialization format Mendix uses to store model elements inside MPR files. Each document (entity, microflow, page, etc.) is stored as a BSON blob.

**Building Block**
A reusable page fragment that can be dragged onto pages in Studio Pro. Similar to a Snippet but intended for the toolbox.

## C

**Catalog**
In mxcli, an in-memory SQLite database that indexes project metadata for fast querying, full-text search, and cross-reference navigation.

**Constant**
A named value defined at the module level that can be configured per environment (development, acceptance, production). Used for API keys, URLs, feature flags, etc.

## D

**DataGrid**
A widget that displays a list of objects in a tabular format with columns, sorting, and search. The classic DataGrid uses server-side rendering; DataGrid2 is the pluggable widget version.

**DataView**
A widget that displays a single object, providing a context for input widgets (TextBox, DatePicker, etc.) to read and write attributes.

**Domain Model**
The data model for a module, consisting of entities, their attributes, and associations between them. Analogous to an entity-relationship diagram.

## E

**Entity**
A data type in the domain model. Equivalent to a table in a relational database. Entities can be persistent (stored in the database), non-persistent (in-memory only), or view (backed by an OQL query).

**Enumeration**
A fixed set of named values (e.g., OrderStatus with values Draft, Active, Closed). Used as attribute types when the set of possible values is known at design time.

**Event Handler**
Logic that runs automatically before or after a commit or delete operation on an entity. Configured on the entity in the domain model.

## G

**Gallery**
A pluggable widget that displays a list of objects in a card/tile layout, as opposed to the tabular layout of a DataGrid.

**Generalization**
Entity inheritance. When entity B generalizes entity A, B inherits all of A's attributes and associations. Analogous to class inheritance in object-oriented programming.

## L

**Layout**
A page template that defines the common structure (header, sidebar, footer) shared by multiple pages. Every page references a layout.

**LayoutGrid**
A responsive grid widget based on a 12-column system. Used to create multi-column layouts with breakpoints for phone, tablet, and desktop.

## M

**MDL (Mendix Definition Language)**
A SQL-like text language for querying and modifying Mendix projects. The primary interface for mxcli.

**Microflow**
Server-side logic expressed as a visual flow of activities. Microflows can retrieve data, call actions, make decisions, loop over lists, and return values. They run on the Mendix runtime (server).

**Module**
A top-level organizational unit in a Mendix project. Each module contains its own domain model, microflows, pages, enumerations, and security settings. Analogous to a package or namespace.

**Module Role**
A permission role defined within a module. Module roles are assigned to user roles at the project level. Entity access rules, microflow access, and page access are granted to module roles.

**MPR**
Mendix Project Resource file. The binary file (`.mpr`) that contains the complete Mendix project model. It is a SQLite database with BSON-encoded documents.

## N

**Nanoflow**
Client-side logic that runs in the browser (or native app). Syntactically similar to microflows but with restrictions (no database transactions, limited activity types). Used for offline-capable and low-latency operations.

## P

**Page**
A user interface screen in a Mendix application. Pages contain widgets (TextBox, DataGrid, Button, etc.) and reference a layout for their outer structure.

**Pluggable Widget**
A widget built with the Mendix Pluggable Widget API (React-based). Examples: DataGrid2, ComboBox, Gallery. Pluggable widgets use JSON-based property schemas (PropertyTypes) stored in the MPR.

## R

**Runtime**
The Mendix server-side execution environment (Java-based) that runs the application. It interprets the model, handles HTTP requests, executes microflows, and manages the database.

## S

**Scheduled Event**
A microflow that runs automatically on a timer (e.g., every hour, daily at midnight). Configured with a start time and interval.

**Snippet**
A reusable UI fragment that can be embedded in multiple pages via a SnippetCall widget. Snippets can accept parameters.

**Studio Pro**
The visual IDE for building Mendix applications. It reads and writes `.mpr` files. mxcli provides a complementary text-based interface to the same project files.

**Storage Name**
The internal type identifier used in BSON `$Type` fields. Often the same as the qualified name, but not always (e.g., `DomainModels$EntityImpl` is the storage name for `DomainModels$Entity`).

## U

**User Role**
A project-level role that aggregates module roles from one or more modules. End users are assigned user roles, which determine their permissions across the entire application.

## W

**Widget**
A UI component on a page. Built-in widgets include TextBox, DataGrid, Button, Container, and LayoutGrid. Pluggable widgets extend this set with third-party components.

**Workflow**
A long-running process with user tasks, decisions, and parallel paths. Workflows model approval processes, multi-step procedures, and human-in-the-loop automation.
