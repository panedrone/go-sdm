SELECT DISTINCT
 [OrderDetails].OrderID,
 [OrderDetails].ProductID,
 Products.ProductName,
 [OrderDetails].UnitPrice,
 [OrderDetails].Quantity,
 [OrderDetails].Discount,
 ([OrderDetails].[UnitPrice]*[Quantity]*(1-[Discount])/100)*100 AS ExtendedPrice
FROM Products INNER JOIN [OrderDetails] ON Products.ProductID = [OrderDetails].ProductID
ORDER BY [OrderDetails].OrderID;
