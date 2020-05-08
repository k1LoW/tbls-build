CREATE TABLE users (
  id serial PRIMARY KEY,
  username varchar (50) UNIQUE NOT NULL,
  password varchar (50) NOT NULL,
  email varchar (355) UNIQUE NOT NULL,
  created timestamp NOT NULL,
  updated timestamp
);
COMMENT ON TABLE users IS 'Users table';
COMMENT ON COLUMN users.email IS 'ex. user@example.com';

CREATE TABLE user_options (
  user_id int PRIMARY KEY,
  show_email boolean NOT NULL DEFAULT false,
  created timestamp NOT NULL,
  updated timestamp
);
COMMENT ON TABLE user_options IS 'User options table';

CREATE TABLE user_access_logs (
  user_id int PRIMARY KEY,
  ua text,
  created timestamp NOT NULL
);

CREATE TABLE posts (
  id bigserial NOT NULL,
  user_id int NOT NULL,
  title varchar (255) NOT NULL,
  body text NOT NULL,
  labels varchar (50)[],
  created timestamp without time zone NOT NULL,
  updated timestamp without time zone,
  CONSTRAINT posts_id_pk PRIMARY KEY(id),
  CONSTRAINT posts_user_id_fk FOREIGN KEY(user_id) REFERENCES users(id) MATCH SIMPLE ON UPDATE NO ACTION ON DELETE CASCADE,
  UNIQUE(user_id, title)
);
COMMENT ON TABLE posts IS 'Posts table';
COMMENT ON COLUMN posts.labels IS 'Posts labels';

CREATE INDEX posts_user_id_idx ON posts USING btree(user_id);

COMMENT ON CONSTRAINT posts_user_id_fk ON posts IS 'posts -> users';

COMMENT ON INDEX posts_user_id_idx IS 'posts.user_id index';

CREATE TABLE comments (
  id bigserial NOT NULL,
  post_id bigint NOT NULL,
  user_id int NOT NULL,
  comment text NOT NULL,
  created timestamp without time zone NOT NULL,
  updated timestamp without time zone,
  CONSTRAINT comments_id_pk PRIMARY KEY(id),
  CONSTRAINT comments_post_id_fk FOREIGN KEY(post_id) REFERENCES posts(id) MATCH SIMPLE,
  CONSTRAINT comments_user_id_fk FOREIGN KEY(user_id) REFERENCES users(id) MATCH SIMPLE,
  UNIQUE(post_id, user_id)
);
COMMENT ON TABLE comments IS E'Comments\nMulti-line\r\ntable\rcomment';
COMMENT ON COLUMN comments.comment IS E'Comment\nMulti-line\r\ncolumn\rcomment';
