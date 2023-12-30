SELECT
    ap.autograph_purchase_id AS "id",
    ap.order_number AS "order",
    ap.product_price AS "productPrice",
    i.name AS "influencerName",
    p.name AS "productName",
    u.email AS "email"
FROM
    autograph_purchaseASap innser
JOIN influencer AS i
    ON autograph_purchase.influencer_id = influencer.influencer_id
LEFT JOIN product AS p
    ON product.product_id = autograph_purchase.product_id
LEFT JOIN USER1 AS u
    ON USER1.user_id = autograph_purchase.user_id
