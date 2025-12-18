package generic

// Generic query to get all org data
// Requires you to provide the logical "WHERE" clause to get org data for specific use case
var QueryOrgData string = `
	SELECT

		o.id,
		o.created_by,
		o.name,
		o.description,
		o.created_at,

		COUNT(DISTINCT om.member_id) AS total_members,
		COALESCE(ARRAY_AGG(DISTINCT om.member_id), '{}') AS members,

		COUNT(DISTINCT p.id) AS total_projects,
		COALESCE(
			JSONB_AGG(
				DISTINCT JSONB_BUILD_OBJECT(
					'id', p.id,
					'name', p.name
				)
			) FILTER (WHERE p.id IS NOT NULL),
			'[]'
		) AS projects,

		COUNT(DISTINCT d.id) AS total_diagrams

	FROM organizations o
	LEFT JOIN projects p ON p.organization_id = o.id
	LEFT JOIN diagrams d ON d.project_id = p.id
	LEFT JOIN organization_members om ON om.organization_id = o.id
	%s
	GROUP BY o.id, o.created_by, o.name, o.description
`
