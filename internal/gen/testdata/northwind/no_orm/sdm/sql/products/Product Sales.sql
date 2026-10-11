SELECT DISTINCT
Categories.CategoryName,
Products.ProductName,
Sum(([OrderDetails].[UnitPrice]*[Quantity]*(1-[Discount])/100)*100) AS ProductSales
FROM (Categories INNER JOIN Products ON Categories.CategoryID = Products.CategoryID) 
	INNER JOIN (Orders INNER JOIN [OrderDetails] ON Orders.OrderID = [OrderDetails].OrderID)
	ON Products.ProductID = [OrderDetails].ProductID
WHERE (((Orders.ShippedDate) Between ? And ?))
GROUP BY Categories.CategoryName, Products.ProductName;
