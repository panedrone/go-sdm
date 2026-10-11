SELECT DISTINCT Orders.ShippedDate, Orders.OrderID, Sum([OrderDetails].UnitPrice * [OrderDetails].Quantity) AS ProductSales, strftime('%Y', [OrderDate]) AS ShippedYear
FROM Orders INNER JOIN [OrderDetails] ON Orders.OrderID = [OrderDetails].OrderID
WHERE (((Orders.ShippedDate) Is Not Null And (Orders.ShippedDate) Between ? And ?));
