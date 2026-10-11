SELECT DISTINCT Categories.CategoryID, Categories.CategoryName, Products.ProductName, Sum([OrderDetails].UnitPrice * [OrderDetails].Quantity) AS ProductSales
FROM Categories INNER JOIN (Products INNER JOIN (Orders INNER JOIN [OrderDetails] ON Orders.OrderID = [OrderDetails].OrderID) ON Products.ProductID = [OrderDetails].ProductID) ON Categories.CategoryID = Products.CategoryID
WHERE (((Orders.OrderDate) Between ? And ?))
GROUP BY Categories.CategoryID, Categories.CategoryName, Products.ProductName
ORDER BY Products.ProductName;
