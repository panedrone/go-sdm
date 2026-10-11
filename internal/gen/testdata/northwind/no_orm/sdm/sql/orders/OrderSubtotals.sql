SELECT [OrderDetails].[OrderID], Sum(([UnitPrice]*[Quantity]*(1-[Discount])/100)*100) AS Subtotal
FROM [OrderDetails]
GROUP BY [OrderDetails].[OrderID];
