CREATE TABLE [Regions](
	[RegionID] integer primary key NOT NULL,
	[RegionDescription] nvarchar(50) NOT NULL COLLATE NOCASE
);

CREATE TABLE [PreviousEmployees](
	[EmployeeID] integer primary key NOT NULL,
	[LastName] nvarchar(20) NOT NULL COLLATE NOCASE,
	[FirstName] nvarchar(10) NOT NULL COLLATE NOCASE,
	[Title] nvarchar(30) NULL COLLATE NOCASE,
	[TitleOfCourtesy] nvarchar(25) NULL COLLATE NOCASE,
	[BirthDate] datetime NULL,
	[HireDate] datetime NULL,
	[Address] nvarchar(60) NULL COLLATE NOCASE,
	[City] nvarchar(15) NULL COLLATE NOCASE,
	[Region] nvarchar(15) NULL COLLATE NOCASE,
	[PostalCode] nvarchar(10) NULL COLLATE NOCASE,
	[Country] nvarchar(15) NULL COLLATE NOCASE,
	[HomePhone] nvarchar(24) NULL COLLATE NOCASE,
	[Extension] nvarchar(4) NULL COLLATE NOCASE,
	[Photo] image NULL,
	[Notes] ntext NULL,
	[PhotoPath] nvarchar(255) NULL COLLATE NOCASE,
    CONSTRAINT [CK_PreviousEmployeesBirthdate] CHECK ([BirthDate]<CURRENT_TIMESTAMP)
);

CREATE TABLE [Employees](
	[EmployeeID] integer primary key autoincrement NOT NULL,
	[LastName] nvarchar(20) NOT NULL COLLATE NOCASE,
	[FirstName] nvarchar(10) NOT NULL COLLATE NOCASE,
	[Title] nvarchar(30) NULL COLLATE NOCASE,
	[TitleOfCourtesy] nvarchar(25) NULL COLLATE NOCASE,
	[BirthDate] datetime NULL,
	[HireDate] datetime NULL,
	[Address] nvarchar(60) NULL COLLATE NOCASE,
	[City] nvarchar(15) NULL COLLATE NOCASE,
	[Region] nvarchar(15) NULL COLLATE NOCASE,
	[PostalCode] nvarchar(10) NULL COLLATE NOCASE,
	[Country] nvarchar(15) NULL COLLATE NOCASE,
	[HomePhone] nvarchar(24) NULL COLLATE NOCASE,
	[Extension] nvarchar(4) NULL COLLATE NOCASE,
	[Photo] image NULL,
	[Notes] ntext NULL,
	[PhotoPath] nvarchar(255) NULL COLLATE NOCASE,
    CONSTRAINT [CK_Birthdate] CHECK  ([BirthDate] < CURRENT_TIMESTAMP)
);

CREATE TABLE [Customers](
	[CustomerID] nchar(5) primary key NOT NULL COLLATE NOCASE,
	[CompanyName] nvarchar(40) NOT NULL COLLATE NOCASE,
	[ContactName] nvarchar(30) NULL COLLATE NOCASE,
	[ContactTitle] nvarchar(30) NULL COLLATE NOCASE,
	[Address] nvarchar(60) NULL COLLATE NOCASE,
	[City] nvarchar(15) NULL COLLATE NOCASE,
	[Region] nvarchar(15) NULL COLLATE NOCASE,
	[PostalCode] nvarchar(10) NULL COLLATE NOCASE,
	[Country] nvarchar(15) NULL COLLATE NOCASE,
	[Phone] nvarchar(24) NULL COLLATE NOCASE,
	[Fax] nvarchar(24) NULL COLLATE NOCASE
);

CREATE TABLE [Suppliers](
	[SupplierID] integer primary key autoincrement NOT NULL,
	[CompanyName] nvarchar(40) NOT NULL COLLATE NOCASE,
	[ContactName] nvarchar(30) NULL COLLATE NOCASE,
	[ContactTitle] nvarchar(30) NULL COLLATE NOCASE,
	[Address] nvarchar(60) NULL COLLATE NOCASE,
	[City] nvarchar(15) NULL COLLATE NOCASE,
	[Region] nvarchar(15) NULL COLLATE NOCASE,
	[PostalCode] nvarchar(10) NULL COLLATE NOCASE,
	[Country] nvarchar(15) NULL COLLATE NOCASE,
	[Phone] nvarchar(24) NULL COLLATE NOCASE,
	[Fax] nvarchar(24) NULL COLLATE NOCASE,
	[HomePage] ntext NULL COLLATE NOCASE
);

CREATE TABLE [InternationalOrders](
	[OrderID] integer primary key NOT NULL,
	[CustomsDescription] nvarchar(100) NOT NULL COLLATE NOCASE,
	[ExciseTax] money NOT NULL,
    CONSTRAINT [FK_InternationalOrders_Orders] FOREIGN KEY([OrderID])
    REFERENCES [Orders] ([OrderID])
);

CREATE TABLE [OrderDetails](
	[OrderID] integer NOT NULL,
	[ProductID] integer NOT NULL,
	[UnitPrice] money NOT NULL CONSTRAINT [DF_Order_Details_UnitPrice]  DEFAULT (0),
	[Quantity] smallint NOT NULL CONSTRAINT [DF_Order_Details_Quantity]  DEFAULT (1),
	[Discount] real NOT NULL CONSTRAINT [DF_Order_Details_Discount]  DEFAULT (0),
    CONSTRAINT [PK_Order_Details] PRIMARY KEY
    (
 	[OrderID] ASC,
 	[ProductID] ASC
    ),
    CONSTRAINT [CK_Discount]  CHECK ([Discount] >= 0 and [Discount] <= 1),
    CONSTRAINT [CK_Quantity]  CHECK ([Quantity] > 0),
    CONSTRAINT [CK_UnitPrice] CHECK ([UnitPrice] >= 0),
    CONSTRAINT [FK_Order_Details_Orders] FOREIGN KEY([OrderID])
    REFERENCES [Orders] ([OrderID]),
    CONSTRAINT [FK_Order_Details_Products] FOREIGN KEY([ProductID])
    REFERENCES [Products] ([ProductID])
);

CREATE TABLE [Territories](
	[TerritoryID] integer primary key NOT NULL,
	[TerritoryDescription] nvarchar(50) NOT NULL COLLATE NOCASE,
	[RegionID] integer NOT NULL,
    CONSTRAINT [FK_Territories_Region] FOREIGN KEY([RegionID])
    REFERENCES [Regions] ([RegionID])
);

CREATE TABLE [EmployeesTerritories](
	[EmployeeID] integer NOT NULL,
	[TerritoryID] integer NOT NULL,
    CONSTRAINT [PK_EmployeeTerritories] PRIMARY KEY 
    (
	[EmployeeID] ASC,
	[TerritoryID] ASC
    ),
    CONSTRAINT [FK_EmployeeTerritories_Employees] FOREIGN KEY([EmployeeID])
    REFERENCES [Employees] ([EmployeeID]),
    CONSTRAINT [FK_EmployeeTerritories_Territories] FOREIGN KEY([TerritoryID])
    REFERENCES [Territories] ([TerritoryID])
);

CREATE TABLE [Products](
	[ProductID] integer primary key autoincrement NOT NULL,
	[ProductName] nvarchar(40) NOT NULL COLLATE NOCASE,
	[SupplierID] integer NULL,
	[CategoryID] integer NULL,
	[QuantityPerUnit] nvarchar(20) NULL COLLATE NOCASE,
	[UnitPrice] money NULL CONSTRAINT [DF_Products_UnitPrice]  DEFAULT (0),
	[UnitsInStock] smallint NULL CONSTRAINT [DF_Products_UnitsInStock]  DEFAULT (0),
	[UnitsOnOrder] smallint NULL CONSTRAINT [DF_Products_UnitsOnOrder]  DEFAULT (0),
	[ReorderLevel] smallint NULL CONSTRAINT [DF_Products_ReorderLevel]  DEFAULT (0),
	[Discontinued] bit NOT NULL CONSTRAINT [DF_Products_Discontinued]  DEFAULT (0),
	[DiscontinuedDate] datetime NULL,
    CONSTRAINT [CK_Products_UnitPrice] CHECK  ([UnitPrice] >= 0),
    CONSTRAINT [CK_ReorderLevel] CHECK  ([ReorderLevel] >= 0),
    CONSTRAINT [CK_UnitsInStock] CHECK  ([UnitsInStock] >= 0),
    CONSTRAINT [CK_UnitsOnOrder] CHECK  ([UnitsOnOrder] >= 0),
    CONSTRAINT [FK_Products_Categories] FOREIGN KEY([CategoryID])
    REFERENCES [Categories] ([CategoryID]),
    CONSTRAINT [FK_Products_Suppliers] FOREIGN KEY([SupplierID])
    REFERENCES [Suppliers] ([SupplierID])
);

CREATE TABLE [Orders](
	[OrderID] integer primary key autoincrement NOT NULL,
	[CustomerID] nchar(5) NULL COLLATE NOCASE,
	[EmployeeID] integer NULL,
	[OrderDate] datetime NULL,
	[RequiredDate] datetime NULL,
	[ShippedDate] datetime NULL,
	[Freight] money NULL CONSTRAINT [DF_Orders_Freight]  DEFAULT (0),
	[ShipName] nvarchar(40) NULL COLLATE NOCASE,
	[ShipAddress] nvarchar(60) NULL COLLATE NOCASE,
	[ShipCity] nvarchar(15) NULL COLLATE NOCASE,
	[ShipRegion] nvarchar(15) NULL COLLATE NOCASE,
	[ShipPostalCode] nvarchar(10) NULL COLLATE NOCASE,
	[ShipCountry] nvarchar(15) NULL COLLATE NOCASE,
    CONSTRAINT [FK_Orders_Customers] FOREIGN KEY([CustomerID])
    REFERENCES [Customers] ([CustomerID])
);

CREATE TABLE [Categories] (
    [CategoryID] integer PRIMARY KEY AUTOINCREMENT NOT NULL,
    [CategoryName] nvarchar(15) NOT NULL COLLATE NOCASE,
    [Description] nvarchar COLLATE NOCASE,
    [Picture] varbinary
);

CREATE INDEX [IX_Employees_LastName] ON [Employees] 
(
	[LastName] ASC
);

CREATE INDEX [IX_Employees_PostalCode] ON [Employees] 
(
	[PostalCode] ASC
);

CREATE INDEX [IX_Customers_City] ON [Customers] 
(
	[City] ASC
);

CREATE INDEX [IX_Customers_CompanyName] ON [Customers] 
(
	[CompanyName] ASC
);

CREATE INDEX [IX_Customers_PostalCode] ON [Customers] 
(
	[PostalCode] ASC
);

CREATE INDEX [IX_Customers_Region] ON [Customers] 
(
	[Region] ASC
);

CREATE INDEX [IX_Suppliers_CompanyName] ON [Suppliers] 
(
	[CompanyName] ASC
);

CREATE INDEX [IX_Suppliers_PostalCode] ON [Suppliers] 
(
	[PostalCode] ASC
);

CREATE INDEX [IX_OrderDetails_OrderID] ON [OrderDetails] 
(
	[OrderID] ASC
);

CREATE INDEX [IX_OrderDetails_OrdersOrder_Details] ON [OrderDetails] 
(
	[OrderID] ASC
);

CREATE INDEX [IX_OrderDetails_ProductID] ON [OrderDetails] 
(
	[ProductID] ASC
);

CREATE INDEX [IX_OrderDetails_ProductsOrder_Details] ON [OrderDetails] 
(
	[ProductID] ASC
);

CREATE INDEX [IX_Products_CategoriesProducts] ON [Products] 
(
	[CategoryID] ASC
);

CREATE INDEX [IX_Products_CategoryID] ON [Products] 
(
	[CategoryID] ASC
);

CREATE INDEX [IX_Products_ProductName] ON [Products] 
(
	[ProductName] ASC
);

CREATE INDEX [IX_Products_SupplierID] ON [Products] 
(
	[SupplierID] ASC
);

CREATE INDEX [IX_Products_SuppliersProducts] ON [Products] 
(
	[SupplierID] ASC
);

CREATE INDEX [IX_Orders_CustomerID] ON [Orders] 
(
	[CustomerID] ASC
);

CREATE INDEX [IX_Orders_CustomersOrders] ON [Orders] 
(
	[CustomerID] ASC
);

CREATE INDEX [IX_Orders_EmployeeID] ON [Orders] 
(
	[EmployeeID] ASC
);

CREATE INDEX [IX_Orders_EmployeesOrders] ON [Orders] 
(
	[EmployeeID] ASC
);

CREATE INDEX [IX_Orders_OrderDate] ON [Orders] 
(
	[OrderDate] ASC
);

CREATE INDEX [IX_Orders_ShippedDate] ON [Orders] 
(
	[ShippedDate] ASC
);

CREATE INDEX [IX_Orders_ShipPostalCode] ON [Orders] 
(
	[ShipPostalCode] ASC
);

CREATE INDEX [IX_Categories_CategoryName] ON [Categories] ([CategoryName]);

