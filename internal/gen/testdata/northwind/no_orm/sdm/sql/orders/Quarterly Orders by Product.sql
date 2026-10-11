SELECT Products.ProductName, Orders.CustomerID, strftime('%Y', [OrderDate]) AS OrderYear
FROM Products INNER JOIN (Orders INNER JOIN [OrderDetails] ON Orders.OrderID = [OrderDetails].OrderID) ON Products.ProductID = [OrderDetails].ProductID
WHERE (((Orders.OrderDate) Between ? And ?))
GROUP BY Products.ProductName, Orders.CustomerID, strftime('%Y', [OrderDate])