CREATE TABLE user_pmcs_community_votes (
    checklist_id UUID NOT NULL,
    voter_uid TEXT NOT NULL,
    direction SMALLINT NOT NULL CHECK (direction IN (-1, 1)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (checklist_id, voter_uid),
    CONSTRAINT fk_user_pmcs_community_votes_source
        FOREIGN KEY (checklist_id)
        REFERENCES user_pmcs_community_sources(checklist_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_user_pmcs_community_votes_voter
        FOREIGN KEY (voter_uid) REFERENCES users(uid)
        ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX user_pmcs_community_votes_voter_idx
    ON user_pmcs_community_votes (voter_uid);
